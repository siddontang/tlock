package lock

import (
	"container/list"
	"fmt"
	"hash/crc32"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultPathSlotSize = 4096

type pendingNode struct {
	path string
	ch   chan struct{}
}

func newPendingNode(path string) *pendingNode {
	n := new(pendingNode)
	n.path = path
	n.ch = make(chan struct{}, 1)
	return n
}

type pathLockerSet struct {
	m sync.Mutex

	set *refValueSet

	pendingLock sync.Mutex
	pendingPath *list.List
}

func newPathLockerSet() *pathLockerSet {
	s := new(pathLockerSet)
	s.pendingPath = list.New()

	f := func(v *refValue) {
		v.v = false
	}

	s.set = newRefValueSet(f)
	return s
}

// a/b/c/ return ["a/", "a/b/", "a/b/c/"]
func (s *pathLockerSet) makeAncestorPaths(path string) []string {
	items := make([]string, 0, 4)

	pos := 0
	for {
		index := strings.IndexByte(path[pos:], '/')
		if index == -1 {
			break
		}

		item := path[0 : pos+index+1]
		items = append(items, item)

		pos += index + 1
		if pos >= len(path) {
			break
		}
	}

	return items
}

func (s *pathLockerSet) tryLock(path string) *pendingNode {
	items := s.makeAncestorPaths(path)

	vs := make([]*refValue, 0, len(items))

	s.m.Lock()
	defer s.m.Unlock()

	var v *refValue
	locked := true

	for _, item := range items {
		v = s.set.Get(item)
		vs = append(vs, v)

		if v.v == true {
			// other lock the ancestor path
			locked = false
			break
		}
	}

	if v.ref != 1 {
		// other lock the children path
		locked = false
	} else {
		v.v = true
		locked = true
	}

	if !locked {
		for i, v := range vs {
			s.set.Put(items[i], v)
		}
		return newPendingNode(path)
	} else {
		return nil
	}
}

func (s *pathLockerSet) LockTimeout(path string, t *time.Timer) bool {
	for {
		n := s.tryLock(path)
		if n == nil {
			return true
		}
		s.addPendingNode(n)

		select {
		case <-n.ch:
		case <-t.C:
			return false
		}
	}
}

func (s *pathLockerSet) addPendingNode(n *pendingNode) {
	s.pendingLock.Lock()
	s.pendingPath.PushBack(n)
	s.pendingLock.Unlock()
}

func (s *pathLockerSet) noticePendingNode(path string) {
	s.pendingLock.Lock()

	var next *list.Element
	for e := s.pendingPath.Front(); e != nil; e = next {
		m := e.Value.(*pendingNode)
		next = e.Next()

		if strings.Contains(path, m.path) || strings.Contains(m.path, path) {
			s.pendingPath.Remove(e)
			m.ch <- struct{}{}
		}
	}

	s.pendingLock.Unlock()
}

func (s *pathLockerSet) Unlock(path string) {
	s.unlock(path)

	s.noticePendingNode(path)
}

func (s *pathLockerSet) unlock(path string) {
	s.m.Lock()
	defer s.m.Unlock()

	items := s.makeAncestorPaths(path)

	for i := len(items) - 1; i >= 0; i-- {
		key := items[i]

		v := s.set.RawGet(key)
		if v == nil {
			panic(fmt.Sprintf("%s is not locked, panic", path))
		}

		v.v = false
		s.set.Put(key, v)
	}
}

type PathLockerGroup struct {
	set []*pathLockerSet
}

func (g *PathLockerGroup) Lock(paths ...string) {
	// use a very long timeout
	b := g.LockTimeout(InfiniteTimeout, paths...)
	if !b {
		panic("Wait lock too long, panic")
	}
}

func (g *PathLockerGroup) LockTimeout(timeout time.Duration, paths ...string) bool {
	if len(paths) == 0 {
		panic("empty paths, panic")
	}

	paths = g.canoicalizePaths(paths...)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	grapNum := 0
	for _, path := range paths {
		b := g.getSet(path).LockTimeout(path, timer)
		if !b {
			g.Unlock(paths[0:grapNum]...)
			return false
		} else {
			grapNum++
		}
	}

	return true
}

func (g *PathLockerGroup) Unlock(paths ...string) {
	if len(paths) == 0 {
		return
	}

	paths = g.canoicalizePaths(paths...)
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))

	for _, path := range paths {
		g.getSet(path).Unlock(path)
	}
}

func (g *PathLockerGroup) getSet(path string) *pathLockerSet {
	base := strings.SplitN(path, "/", 2)
	index := crc32.ChecksumIEEE([]byte(base[0])) % uint32(defaultPathSlotSize)
	return g.set[index]
}

func (g *PathLockerGroup) canonicalizePath(p string) string {
	p = path.Clean(p)

	// remove first, so /a/b/c will be a/b/c
	p = strings.TrimPrefix(p, "/")

	// add / suffix, path Clean will remove the / suffix
	p = p + "/"

	return p
}

func (g *PathLockerGroup) canoicalizePaths(paths ...string) []string {
	p := make([]string, 0, len(paths))

	for i, path := range paths {
		paths[i] = g.canonicalizePath(path)
		if paths[i] == "/" {
			panic("invalid path, can not empty")
		}
	}

	sort.Strings(paths)

	p = append(p, paths[0])

	for i := 1; i < len(paths); i++ {
		for j := 0; j < len(p); j++ {
			if strings.Contains(paths[i], p[j]) {
				// if we want to lock a/b and a/b/c at same time, we only
				// need to lock the parent path a/b
				break
			}
		}

		p = append(p, paths[i])
	}

	return p
}

func NewPathLockerGroup() *PathLockerGroup {
	g := new(PathLockerGroup)
	g.set = make([]*pathLockerSet, defaultPathSlotSize)
	for i := 0; i < defaultPathSlotSize; i++ {
		g.set[i] = newPathLockerSet()
	}

	return g
}

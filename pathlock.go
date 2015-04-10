package tlock

import (
	"fmt"
	"hash/crc32"
	"path"
	"sort"
	"strings"
	"time"
)

const defaultPathSlotSize = 4096

type PathLockerGroup struct {
	set []*refLockSet
}

// a/b/c/ return ["a/", "a/b/", "a/b/c/"]
func makeAncestorPaths(path string) []string {
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

	grapPathNum := 0

	for _, path := range paths {
		items := makeAncestorPaths(path)

		s := g.getSet(path)
		grapLockNum := 0

		for i, item := range items {
			m := s.Get(item)
			var b bool
			if i == len(items)-1 {
				// final node, use write lock
				b = LockWithTimer(m, timer)
			} else {
				// ntermediate node, use read lock
				b = LockWithTimer(m.RLocker(), timer)
			}

			if !b {
				s.Put(item, m)

				g.unlockPathItems(s, items[0:grapLockNum], false)
				g.Unlock(paths[0:grapPathNum]...)

				return false
			} else {
				grapLockNum++
			}
		}

		grapPathNum++
	}

	return true
}

func (g *PathLockerGroup) unlockPathItems(s *refLockSet, items []string, finalIsWriteLock bool) {
	for i := len(items) - 1; i >= 0; i-- {
		m := s.RawGet(items[i])
		if m == nil {
			panic(fmt.Sprintf("%s is not locked, panic", items[i]))
		}

		if finalIsWriteLock && i == len(items)-1 {
			// final node, use write lock
			m.Unlock()
		} else {
			// intermediate node, use read lock
			m.RUnlock()
		}

		s.Put(items[i], m)
	}
}

func (g *PathLockerGroup) Unlock(paths ...string) {
	if len(paths) == 0 {
		return
	}

	paths = g.canoicalizePaths(paths...)
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))

	for _, path := range paths {
		items := makeAncestorPaths(path)

		s := g.getSet(path)

		g.unlockPathItems(s, items, true)
	}
}

func (g *PathLockerGroup) getSet(path string) *refLockSet {
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
	g.set = make([]*refLockSet, defaultPathSlotSize)
	for i := 0; i < defaultPathSlotSize; i++ {
		g.set[i] = newRefLockSet()
	}

	return g
}

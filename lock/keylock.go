package lock

import (
	"fmt"
	"hash/crc32"
	"sort"
	"sync"
	"time"
)

const defaultKeySlotSize = 1024

type keyLockerSet struct {
	m sync.Mutex

	set *refValueSet
}

func (s *keyLockerSet) Get(key string) *refValue {
	s.m.Lock()
	r := s.set.Get(key)
	s.m.Unlock()
	return r
}

func (s *keyLockerSet) RawGet(key string) *refValue {
	s.m.Lock()
	r := s.set.RawGet(key)
	s.m.Unlock()
	return r
}

func (s *keyLockerSet) Put(key string, v *refValue) {
	s.m.Lock()
	s.set.Put(key, v)
	s.m.Unlock()
}

func newKeyLockerSet() *keyLockerSet {
	s := new(keyLockerSet)
	f := func(v *refValue) {
		value := make(chan struct{}, 1)
		value <- struct{}{}
		v.v = value
	}

	s.set = newRefValueSet(f)
	return s
}

type KeyLockerGroup struct {
	set []*keyLockerSet
}

func NewKeyLockerGroup() *KeyLockerGroup {
	g := new(KeyLockerGroup)

	g.set = make([]*keyLockerSet, defaultKeySlotSize)
	for i := 0; i < defaultKeySlotSize; i++ {
		g.set[i] = newKeyLockerSet()
	}
	return g

}

func (g *KeyLockerGroup) getSet(key string) *keyLockerSet {
	index := crc32.ChecksumIEEE([]byte(key)) % uint32(defaultKeySlotSize)
	return g.set[index]
}

func (g *KeyLockerGroup) Lock(keys ...string) {
	// use a very long timeout
	b := g.LockTimeout(InfiniteTimeout, keys...)
	if !b {
		panic("Wait lock too long, panic")
	}
}

func (g *KeyLockerGroup) LockTimeout(timeout time.Duration, keys ...string) bool {
	if len(keys) == 0 {
		panic("empty keys, panic")
	}

	// Sort keys to avoid deadlock
	sort.Strings(keys)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	grapNum := 0

	for _, key := range keys {
		v := g.getSet(key).Get(key)
		ch := v.v.(chan struct{})
		select {
		case <-ch:
			// grap the lock
			grapNum++
		case <-timer.C:
			g.getSet(key).Put(key, v)
			g.Unlock(keys[0:grapNum]...)
			return false
		}
	}
	return true
}

func (g *KeyLockerGroup) Unlock(keys ...string) {
	if len(keys) == 0 {
		return
	}

	// Reverse Sort keys to avoid deadlock
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	for _, key := range keys {
		v := g.getSet(key).RawGet(key)

		if v == nil {
			panic(fmt.Sprintf("%s is not locked, panic", key))
		}

		ch := v.v.(chan struct{})
		select {
		case ch <- struct{}{}:
		default:
			panic(fmt.Sprintf("%s is not locked, panic", key))
		}

		g.getSet(key).Put(key, v)
	}
}

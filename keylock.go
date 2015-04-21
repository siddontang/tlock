package tlock

import (
	"fmt"
	"hash/crc32"
	"sort"
	"time"
)

const defaultKeySlotSize = 1024

type KeyLockerGroup struct {
	set []*refLockSet
}

func NewKeyLockerGroup() *KeyLockerGroup {
	g := new(KeyLockerGroup)

	g.set = make([]*refLockSet, defaultKeySlotSize)
	for i := 0; i < defaultKeySlotSize; i++ {
		g.set[i] = newRefLockSet()
	}
	return g

}

func (g *KeyLockerGroup) getSet(key string) *refLockSet {
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

func removeDuplicatedItems(keys ...string) []string {
	if len(keys) <= 1 {
		return keys
	}

	m := make(map[string]struct{}, len(keys))

	p := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, ok := m[key]; !ok {
			m[key] = struct{}{}
			p = append(p, key)
		}
	}

	return p
}

func (g *KeyLockerGroup) LockTimeout(timeout time.Duration, keys ...string) bool {
	if len(keys) == 0 {
		panic("empty keys, panic")
	}

	// remove duplicated items
	keys = removeDuplicatedItems(keys...)

	// Sort keys to avoid deadlock
	sort.Strings(keys)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	grapNum := 0

	for _, key := range keys {
		s := g.getSet(key)
		m := s.Get(key)
		b := LockWithTimer(m, timer)
		if !b {
			s.Put(key, m)
			g.Unlock(keys[0:grapNum]...)
			return false
		} else {
			grapNum++
		}
	}
	return true
}

func (g *KeyLockerGroup) Unlock(keys ...string) {
	if len(keys) == 0 {
		return
	}

	// remove duplicated items
	keys = removeDuplicatedItems(keys...)

	// Reverse Sort keys to avoid deadlock
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	for _, key := range keys {
		m := g.getSet(key).RawGet(key)

		if m == nil {
			panic(fmt.Sprintf("%s is not locked, panic", key))
		}

		m.Unlock()

		g.getSet(key).Put(key, m)
	}
}

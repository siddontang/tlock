package lock

import (
	"fmt"
	"sort"
	"time"
)

type KeyLockerGroup struct {
	set *refValueSet
}

func NewKeyLockerGroup() *KeyLockerGroup {
	g := new(KeyLockerGroup)

	f := func(v *refValue) {
		value := make(chan struct{}, 1)
		value <- struct{}{}
		v.v = value
	}

	g.set = newRefValueSet(f)

	return g
}

func (g *KeyLockerGroup) Lock(keys ...string) {
	// use a very long timeout
	b := g.LockTimeout(30*24*3600*time.Second, keys...)
	if !b {
		panic("Wait lock too long, panic")
	}
}

func (g *KeyLockerGroup) LockTimeout(timeout time.Duration, keys ...string) bool {
	// Sort keys to avoid deadlock
	sort.Strings(keys)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	grapKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		v := g.set.Get(key)
		ch := v.v.(chan struct{})
		select {
		case <-ch:
			// grap the lock
			grapKeys = append(grapKeys, key)
		case <-timer.C:
			g.set.Put(key, v)
			g.Unlock(grapKeys...)
			return false
		}
	}
	return true
}

func (g *KeyLockerGroup) Unlock(keys ...string) {
	// Reverse Sort keys to avoid deadlock
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	for _, key := range keys {
		v := g.set.RawGet(key)

		if v == nil {
			panic(fmt.Sprintf("%s is not locked, panic", key))
		}

		ch := v.v.(chan struct{})
		select {
		case ch <- struct{}{}:
		default:
			panic(fmt.Sprintf("%s is not locked, panic", key))
		}

		g.set.Put(key, v)
	}
}

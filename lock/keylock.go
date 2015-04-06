package lock

import (
	"sort"
	"time"
)

type KeyLocker struct {
	keys  []string
	locks []*refValue

	set *refValueSet

	locked int32
	ch     chan struct{}
}

func (g *KeyLockerGroup) newKeyLocker(keys []string) *KeyLocker {
	// sort keys to avoid dead lock
	sort.Strings(keys)

	l := new(KeyLocker)

	l.keys = keys
	l.locks = make([]*refValue, len(l.keys))
	l.set = g.set

	l.ch = make(chan struct{}, 1)
	l.ch <- struct{}{}

	return l
}

func (l *KeyLocker) Lock() {
	// use a very long timeout
	b := l.LockTimeout(30 * 24 * 3600 * time.Second)
	if !b {
		panic("Wait lock too long, panic")
	}
}

func (l *KeyLocker) LockTimeout(timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-l.ch:
	case <-timer.C:
		return false
	}

	for i, key := range l.keys {
		v := l.set.Get(key)
		ch := v.v.(chan struct{})
		select {
		case <-ch:
			// grap the lock
			l.locks[i] = v
		case <-timer.C:
			l.set.Put(key, v)
			l.Unlock()
			return false
		}
	}
	return true
}

func (l *KeyLocker) Unlock() {
	for i := len(l.locks) - 1; i >= 0; i-- {
		v := l.locks[i]
		if v == nil {
			continue
		}
		ch := v.v.(chan struct{})
		ch <- struct{}{}
		l.set.Put(l.keys[i], v)
		l.locks[i] = nil
	}

	select {
	case l.ch <- struct{}{}:
	default:
		panic("Not locked, panic")
	}
}

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

func (g *KeyLockerGroup) GetLocker(keys ...string) Locker {
	return g.newKeyLocker(keys)
}

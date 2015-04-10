package tlock

import (
	"sync"
)

type refLock struct {
	sync.RWMutex
	ref int
}

type refLockSet struct {
	sync.Mutex
	set map[string]*refLock
}

func newRefLockSet() *refLockSet {
	s := new(refLockSet)

	s.set = make(map[string]*refLock, 16)

	return s
}

func (s *refLockSet) Get(key string) *refLock {
	s.Lock()
	defer s.Unlock()

	v, ok := s.set[key]
	if ok {
		v.ref++
	} else {
		v = &refLock{ref: 1}

		s.set[key] = v
	}

	return v
}

func (s *refLockSet) RawGet(key string) *refLock {
	s.Lock()
	defer s.Unlock()

	v := s.set[key]
	return v
}

func (s *refLockSet) Put(key string, v *refLock) {
	s.Lock()
	defer s.Unlock()

	v.ref--
	if v.ref <= 0 {
		delete(s.set, key)
	}
}

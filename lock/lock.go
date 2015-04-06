package lock

import (
	"sync"
	"time"
)

type Locker interface {
	Lock()
	LockTimeout(timeout time.Duration) bool
	Unlock()
}

type LockerGroup interface {
	GetLocker(args ...string) Locker
}

type refValue struct {
	ref int
	v   interface{}
}

type refValueSet struct {
	sync.Mutex

	initFunc func(*refValue)
	set      map[string]*refValue
}

func newRefValueSet(initFunc func(*refValue)) *refValueSet {
	s := new(refValueSet)

	s.set = make(map[string]*refValue, 1024)
	s.initFunc = initFunc

	return s
}

func (s *refValueSet) Get(key string) *refValue {
	s.Lock()
	defer s.Unlock()

	v, ok := s.set[key]
	if ok {
		v.ref++
	} else {
		v = &refValue{ref: 1}
		s.initFunc(v)

		s.set[key] = v
	}

	return v
}

func (s *refValueSet) Put(key string, v *refValue) {
	s.Lock()
	defer s.Unlock()

	v.ref--
	if v.ref <= 0 {
		delete(s.set, key)
	}
}

func (s *refValueSet) Exists(key string) bool {
	s.Lock()
	defer s.Unlock()

	_, ok := s.set[key]
	return ok
}

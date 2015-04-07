package lock

import (
	"sync"
	"time"
)

type LockerGroup interface {
	Lock(args ...string)
	LockTimeout(timeout time.Duration, args ...string) bool
	Unlock(args ...string)
}

var InfiniteTimeout = 30 * 24 * 3600 * time.Second

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

func (s *refValueSet) RawGet(key string) *refValue {
	s.Lock()
	defer s.Unlock()

	v := s.set[key]
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

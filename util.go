package tlock

import (
	"sync"
	"sync/atomic"
	"time"
)

func LockTimeout(m sync.Locker, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	return LockWithTimer(m, timer)
}

func LockWithTimer(m sync.Locker, timer *time.Timer) bool {
	done := make(chan bool, 1)
	decided := new(int32)
	go func() {
		m.Lock()
		if atomic.SwapInt32(decided, 1) == 0 {
			done <- true
		} else {
			// If we already decided the result, and this thread did not win
			m.Unlock()
		}
	}()
	select {
	case <-done:
		return true
	case <-timer.C:
		if atomic.SwapInt32(decided, 1) == 1 {
			// The other thread already decided the result
			return true
		}
		return false
	}
}

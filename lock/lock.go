package lock

import (
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

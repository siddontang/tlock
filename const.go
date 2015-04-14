package tlock

import (
	"time"
)

type LockerGroup interface {
	Lock(args ...string)
	LockTimeout(timeout time.Duration, args ...string) bool
	Unlock(args ...string)
}

var InfiniteTimeout = 30 * 24 * 3600 * time.Second

const (
	KeyLockType  = "key"
	PathLockType = "path"
)

type Client interface {
	GetLocker(tp string, names ...string) (ClientLocker, error)
}

type ClientLocker interface {
	Lock() error
	// timeout is seconds
	LockTimeout(timeout int) error
	Unlock() error
}

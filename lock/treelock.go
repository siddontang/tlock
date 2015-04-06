package lock

import (
	"time"
)

type TreeLocker struct {
}

func (l *TreeLocker) Lock() {

}

func (l *TreeLocker) LockTimeout(timeout time.Duration) bool {
	return false
}

func (l *TreeLocker) Unlock() {

}

type TreeLockerGroup struct {
}

func (g *TreeLockerGroup) GetLocker(paths ...string) Locker {
	return nil
}

func NewTreeLockerGroup() *TreeLockerGroup {
	return nil
}

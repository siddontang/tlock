package lock

import (
	"time"
)

type TreeLockerGroup struct {
}

func (g *TreeLockerGroup) Lock(paths ...string) {
}

func (g *TreeLockerGroup) LockTimeout(timeout time.Duration, paths ...string) bool {
	return false
}

func (g *TreeLockerGroup) Unlock(paths ...string) {
}

func NewTreeLockerGroup() *TreeLockerGroup {
	return nil
}

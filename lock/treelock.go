package lock

import (
	"time"
)

type TreeLockerGroup struct {
}

func (g *TreeLockerGroup) Lock(paths ...string) {
	// use a very long timeout
	b := g.LockTimeout(InfiniteTimeout, paths...)
	if !b {
		panic("Wait lock too long, panic")
	}
}

func (g *TreeLockerGroup) LockTimeout(timeout time.Duration, paths ...string) bool {
	return false
}

func (g *TreeLockerGroup) Unlock(paths ...string) {
}

func NewTreeLockerGroup() *TreeLockerGroup {
	return nil
}

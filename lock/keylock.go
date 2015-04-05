package lock

import (
	"time"
)

type KeyLocker struct {
	keys []string
}

func (l *KeyLocker) Lock() {

}

func (l *KeyLocker) LockTimeout(timeout time.Duration) bool {
	return true
}

func (l *KeyLocker) Unlock() {

}

type KeyLockerGroup struct {
}

func (g *KeyLockerGroup) GetLocker(keys ...string) Locker {
	return nil
}

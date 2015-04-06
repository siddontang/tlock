package lock

import (
	"sync"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type lcokTestSuite struct {
}

var _ = Suite(&lcokTestSuite{})

func (s *lcokTestSuite) TestKeyLock(c *C) {
	g := NewKeyLockerGroup()

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()
		l := g.GetLocker("a", "b")
		l.Lock()
		l.Unlock()
	}()

	time.Sleep(1 * time.Second)
	l := g.GetLocker("b", "a")
	l.Lock()
	l.Unlock()

	wg.Wait()

	c.Assert(g.set.Exists("a"), Equals, false)
	c.Assert(g.set.Exists("b"), Equals, false)

	wg.Add(1)

	go func() {
		defer wg.Done()

		l := g.GetLocker("a")
		l.Lock()

		time.Sleep(2 * time.Second)

		l.Unlock()
	}()

	time.Sleep(1 * time.Second)

	l = g.GetLocker("a")
	b := l.LockTimeout(100 * time.Millisecond)
	c.Assert(b, Equals, false)
	wg.Wait()

	c.Assert(g.set.Exists("a"), Equals, false)

	b = l.LockTimeout(100 * time.Millisecond)
	c.Assert(b, Equals, true)

	c.Assert(g.set.Exists("a"), Equals, true)

	l.Unlock()

	c.Assert(g.set.Exists("a"), Equals, false)

	wg.Add(1)

	l = g.GetLocker("a")

	go func() {
		defer wg.Done()

		l.Lock()

		time.Sleep(2 * time.Second)

		l.Unlock()
	}()

	time.Sleep(1 * time.Second)

	b = l.LockTimeout(100 * time.Millisecond)
	c.Assert(b, Equals, false)

	wg.Wait()

	c.Assert(g.set.Exists("a"), Equals, false)

	b = l.LockTimeout(100 * time.Millisecond)
	c.Assert(b, Equals, true)

	c.Assert(g.set.Exists("a"), Equals, true)

	l.Unlock()
	c.Assert(g.set.Exists("a"), Equals, false)
}

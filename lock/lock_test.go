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

type lockTestSuite struct {
}

var _ = Suite(&lockTestSuite{})

func (s *lockTestSuite) TestKeyLock(c *C) {
	g := NewKeyLockerGroup()

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()
		g.Lock("a", "b")
		g.Unlock("a", "b")
	}()

	time.Sleep(1 * time.Second)
	g.Lock("b", "a")
	g.Unlock("a", "b")

	wg.Wait()

	c.Assert(g.set.Exists("a"), Equals, false)
	c.Assert(g.set.Exists("b"), Equals, false)

	wg.Add(1)

	go func() {
		defer wg.Done()

		g.Lock("a")

		time.Sleep(2 * time.Second)

		g.Unlock("a")
	}()

	time.Sleep(1 * time.Second)

	b := g.LockTimeout(100*time.Millisecond, "a")
	c.Assert(b, Equals, false)
	wg.Wait()

	c.Assert(g.set.Exists("a"), Equals, false)

	b = g.LockTimeout(100*time.Millisecond, "a")
	c.Assert(b, Equals, true)

	c.Assert(g.set.Exists("a"), Equals, true)

	g.Unlock("a")

	c.Assert(g.set.Exists("a"), Equals, false)

	wg.Add(1)

	go func() {
		defer wg.Done()

		g.Lock("a")

		time.Sleep(2 * time.Second)

		g.Unlock("a")
	}()

	time.Sleep(1 * time.Second)

	b = g.LockTimeout(100*time.Millisecond, "a")
	c.Assert(b, Equals, false)

	wg.Wait()

	c.Assert(g.set.Exists("a"), Equals, false)

	b = g.LockTimeout(100*time.Millisecond, "a")
	c.Assert(b, Equals, true)

	c.Assert(g.set.Exists("a"), Equals, true)

	g.Unlock("a")
	c.Assert(g.set.Exists("a"), Equals, false)
}

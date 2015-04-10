package tlock

import (
	"sync"
	"time"

	. "gopkg.in/check.v1"
)

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

	b = g.LockTimeout(100*time.Millisecond, "a")
	c.Assert(b, Equals, true)

	g.Unlock("a")

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

	b = g.LockTimeout(100*time.Millisecond, "a")
	c.Assert(b, Equals, true)

	g.Unlock("a")
}

func (s *lockTestSuite) TestPathLock(c *C) {
	g := NewPathLockerGroup()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		g.Lock("a/b")
		g.Unlock("a/b")
	}()

	time.Sleep(1 * time.Second)
	g.Lock("a/b")
	g.Unlock("a/b")

	wg.Wait()

	wg.Add(1)

	go func() {
		defer wg.Done()

		g.Lock("a")

		time.Sleep(2 * time.Second)

		g.Unlock("a")
	}()

	time.Sleep(1 * time.Second)

	b := g.LockTimeout(100*time.Millisecond, "a/b")
	c.Assert(b, Equals, false)
	wg.Wait()

	b = g.LockTimeout(100*time.Millisecond, "a/b")
	c.Assert(b, Equals, true)

	g.Unlock("a/b")

	wg.Add(1)

	go func() {
		defer wg.Done()

		g.Lock("a/b")

		time.Sleep(2 * time.Second)

		g.Unlock("a/b")
	}()

	time.Sleep(1 * time.Second)

	b = g.LockTimeout(100*time.Millisecond, "a")
	c.Assert(b, Equals, false)

	wg.Wait()

	b = g.LockTimeout(100*time.Millisecond, "a")
	c.Assert(b, Equals, true)

	g.Unlock("a")

	g.Lock("a/b/c")
	g.Lock("a/b/d")

	g.Unlock("a/b/c")
	g.Unlock("a/b/d")
}

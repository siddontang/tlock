package tlock

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/siddontang/goredis"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type serverTestSuite struct {
	a *App
}

var _ = Suite(&serverTestSuite{})

func (s *serverTestSuite) SetUpSuite(c *C) {
	s.a = NewApp()

	err := s.a.StartHTTP("127.0.0.1:0")
	c.Assert(err, IsNil)

	err = s.a.StartRESP("127.0.0.1:0")
	c.Assert(err, IsNil)
}

func (s *serverTestSuite) TearDownSuite(c *C) {
	if s.a != nil {
		s.a.Close()
	}
}

func (s *serverTestSuite) getLocks(c *C) string {
	c.Assert(s.a.httpListener, NotNil)
	addr := s.a.httpListener.Addr()

	r, err := http.Get(fmt.Sprintf("http://%s/lock", addr))
	c.Assert(err, IsNil)

	defer r.Body.Close()
	c.Assert(r.StatusCode, Equals, http.StatusOK)
	b, err := ioutil.ReadAll(r.Body)
	c.Assert(err, IsNil)
	return string(b)
}

func (s *serverTestSuite) lock(c *C, names string, tp string, timeout int) uint64 {
	c.Assert(s.a.httpListener, NotNil)
	addr := s.a.httpListener.Addr()

	r, err := http.Post(fmt.Sprintf("http://%s/lock?names=%s&type=%s&timeout=%d", addr, url.QueryEscape(names), tp, timeout), "", strings.NewReader(""))
	c.Assert(err, IsNil)

	defer r.Body.Close()
	buf, err := ioutil.ReadAll(r.Body)
	c.Assert(err, IsNil)

	if timeout == 0 {
		c.Assert(r.StatusCode, Equals, http.StatusOK)
		id, err := strconv.ParseUint(string(buf), 10, 64)
		c.Assert(err, IsNil)
		return id
	} else {
		c.Assert(r.StatusCode, Equals, http.StatusRequestTimeout)
		return 0
	}
}

func (s *serverTestSuite) unlock(c *C, id uint64) {
	c.Assert(s.a.httpListener, NotNil)
	addr := s.a.HTTPAddr()

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("http://%s/lock?id=%d", addr, id), nil)
	r, err := http.DefaultClient.Do(req)
	c.Assert(err, IsNil)

	defer r.Body.Close()
	ioutil.ReadAll(r.Body)

	c.Assert(r.StatusCode, Equals, http.StatusOK)
}

func (s *serverTestSuite) TestKeyLock(c *C) {
	var wg sync.WaitGroup

	wg.Add(1)

	names := "a,b"
	tp := "key"

	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)
		id := s.lock(c, names, tp, 0)
		s.unlock(c, id)
	}()

	id := s.lock(c, names, tp, 0)
	s.unlock(c, id)

	wg.Wait()
}

func (s *serverTestSuite) TestPathLock(c *C) {
	var wg sync.WaitGroup

	wg.Add(1)

	names := "a/b,a/c"
	tp := "path"

	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)
		id := s.lock(c, names, tp, 0)
		s.unlock(c, id)
	}()

	id := s.lock(c, names, tp, 0)
	s.unlock(c, id)

	wg.Wait()

	wg.Add(1)

	done := make(chan struct{})
	go func() {
		defer wg.Done()
		id := s.lock(c, names, tp, 0)
		done <- struct{}{}
		time.Sleep(2 * time.Second)
		done <- struct{}{}

		s.unlock(c, id)
	}()

	<-done
	id = s.lock(c, names, tp, 1)
	<-done

	id = s.lock(c, names, tp, 0)
	s.unlock(c, id)
	wg.Wait()
}

func (s *serverTestSuite) TestGetLock(c *C) {
	names := "a/b"
	tp := "key"

	id := s.lock(c, names, tp, 0)
	str := s.getLocks(c)
	c.Assert(strings.Contains(str, names), Equals, true)

	s.unlock(c, id)
	str = s.getLocks(c)
	c.Assert(strings.Contains(str, names), Equals, false)
}

func (s *serverTestSuite) TestRESPLock(c *C) {
	addr := s.a.RESPAddr()
	c.Assert(addr, NotNil)

	c1, err := goredis.Connect(addr.String())
	c.Assert(addr, NotNil)
	defer c1.Close()

	c2, err := goredis.Connect(addr.String())
	c.Assert(addr, NotNil)
	defer c2.Close()

	var wg sync.WaitGroup

	wg.Add(1)

	done := make(chan struct{})
	go func() {
		defer wg.Done()
		id, err := goredis.Bytes(c2.Do("LOCK", "a", "TYPE", "KEY", "TIMEOUT", 0))
		c.Assert(err, IsNil)

		done <- struct{}{}
		time.Sleep(2 * time.Second)
		done <- struct{}{}

		_, err = c2.Do("UNLOCK", id)
		c.Assert(err, IsNil)
	}()

	<-done
	_, err = c1.Do("LOCK", "a", "TYPE", "KEY", "TIMEOUT", 1)
	<-done

	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), errLockTimeout.Error()), Equals, true)

	id, err := goredis.Bytes(c1.Do("LOCK", "a", "TYPE", "KEY", "TIMEOUT", 0))
	c.Assert(err, IsNil)
	_, err = c1.Do("UNLOCK", id)
	c.Assert(err, IsNil)

	wg.Wait()
}

func (s *serverTestSuite) TestRESPLockClose(c *C) {
	addr := s.a.RESPAddr()
	c.Assert(addr, NotNil)

	c1, err := goredis.Connect(addr.String())
	c.Assert(addr, NotNil)
	defer c1.Close()

	c2, err := goredis.Connect(addr.String())
	c.Assert(addr, NotNil)
	defer c2.Close()

	var wg sync.WaitGroup

	wg.Add(1)

	done := make(chan struct{})
	go func() {
		defer wg.Done()
		_, err := goredis.Bytes(c2.Do("LOCK", "a", "TYPE", "KEY", "TIMEOUT", 0))
		c.Assert(err, IsNil)

		done <- struct{}{}
		time.Sleep(2 * time.Second)
		done <- struct{}{}

		c2.Close()
	}()

	<-done
	_, err = c1.Do("LOCK", "a", "TYPE", "KEY", "TIMEOUT", 1)
	<-done

	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), errLockTimeout.Error()), Equals, true)

	id, err := goredis.Bytes(c1.Do("LOCK", "a", "TYPE", "KEY", "TIMEOUT", 0))
	c.Assert(err, IsNil)
	_, err = c1.Do("UNLOCK", id)
	c.Assert(err, IsNil)

	wg.Wait()
}

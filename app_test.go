package tlock

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

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

func (s *serverTestSuite) lock(c *C, names string, tp string, timeout int) {
	c.Assert(s.a.httpListener, NotNil)
	addr := s.a.httpListener.Addr()

	r, err := http.Post(fmt.Sprintf("http://%s/lock?names=%s&type=%s&timeout=%d", addr, url.QueryEscape(names), tp, timeout), "", strings.NewReader(""))
	c.Assert(err, IsNil)

	defer r.Body.Close()
	ioutil.ReadAll(r.Body)

	if timeout == 0 {
		c.Assert(r.StatusCode, Equals, http.StatusOK)
	} else {
		c.Assert(r.StatusCode, Equals, http.StatusRequestTimeout)
	}
}

func (s *serverTestSuite) unlock(c *C, names string, tp string) {
	c.Assert(s.a.httpListener, NotNil)
	addr := s.a.HTTPAddr()

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("http://%s/lock?names=%s&type=%s", addr, url.QueryEscape(names), tp), nil)
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
		s.lock(c, names, tp, 0)
		s.unlock(c, names, tp)
	}()

	s.lock(c, names, tp, 0)
	s.unlock(c, names, tp)

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
		s.lock(c, names, tp, 0)
		s.unlock(c, names, tp)
	}()

	s.lock(c, names, tp, 0)
	s.unlock(c, names, tp)

	wg.Wait()

	wg.Add(1)

	done := make(chan struct{})
	go func() {
		defer wg.Done()
		s.lock(c, names, tp, 0)
		done <- struct{}{}
		time.Sleep(2 * time.Second)
		done <- struct{}{}

		s.unlock(c, names, tp)
	}()

	<-done
	s.lock(c, names, tp, 1)
	<-done

	s.lock(c, names, tp, 0)
	s.unlock(c, names, tp)
	wg.Wait()
}

func (s *serverTestSuite) TestGetLock(c *C) {
	names := "a/b,a/c"
	tp := "path"

	s.lock(c, names, tp, 0)
	str := s.getLocks(c)
	c.Assert(strings.Contains(str, names), Equals, true)

	s.unlock(c, names, tp)
	str = s.getLocks(c)
	c.Assert(strings.Contains(str, names), Equals, false)
}

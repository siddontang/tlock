package server

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

func (s *serverTestSuite) lock(c *C, keys string, tp string) {
	c.Assert(s.a.httpListener, NotNil)
	addr := s.a.httpListener.Addr()

	r, err := http.Post(fmt.Sprintf("http://%s/lock?keys=%s&type=%s", addr, url.QueryEscape(keys), tp), "", strings.NewReader(""))
	c.Assert(err, IsNil)

	defer r.Body.Close()
	ioutil.ReadAll(r.Body)
}

func (s *serverTestSuite) unlock(c *C, keys string, tp string) {
	c.Assert(s.a.httpListener, NotNil)
	addr := s.a.httpListener.Addr()

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("http://%s/lock?keys=%s&type=%s", addr, url.QueryEscape(keys), tp), nil)
	r, err := http.DefaultClient.Do(req)
	c.Assert(err, IsNil)

	defer r.Body.Close()
	ioutil.ReadAll(r.Body)
}

func (s *serverTestSuite) TestKeyLock(c *C) {
	var wg sync.WaitGroup

	wg.Add(1)

	keys := "a,b"
	tp := "key"

	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)
		s.lock(c, keys, tp)
		s.unlock(c, keys, tp)
	}()

	s.lock(c, keys, tp)
	s.unlock(c, keys, tp)

	wg.Wait()
}

func (s *serverTestSuite) TestPathLock(c *C) {
	var wg sync.WaitGroup

	wg.Add(1)

	keys := "a/b"
	tp := "path"

	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)
		s.lock(c, keys, tp)
		s.unlock(c, keys, tp)
	}()

	s.lock(c, keys, tp)
	s.unlock(c, keys, tp)

	wg.Wait()
}

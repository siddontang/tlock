package tlock

import (
	"fmt"
	"strings"

	"github.com/siddontang/goredis"
)

type respLocker struct {
	c     *goredis.Client
	conn  *goredis.PoolConn
	names []string
	tp    string
	id    []byte
}

func NewRESPLocker(c *goredis.Client, tp string, names ...string) (ClientLocker, error) {
	tp = strings.ToLower(tp)
	if tp != "key" && tp != "path" {
		return nil, fmt.Errorf("invalid lock type %s, must key or path", tp)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("empty lock names")
	}

	l := new(respLocker)
	l.c = c
	l.names = names
	l.tp = tp

	return l, nil
}

func (l *respLocker) Lock() error {
	return l.LockTimeout(60)
}

func (l *respLocker) LockTimeout(timeout int) error {
	if l.id != nil {
		return fmt.Errorf("lockid %s exists, must unlock first", l.id)
	}

	conn, err := l.c.Get()
	if err != nil {
		return err
	}

	v := make([]interface{}, 0, len(l.names)+4)
	for _, name := range l.names {
		v = append(v, name)
	}

	v = append(v, "TYPE", l.tp)
	v = append(v, "TIMEOUT", timeout)

	id, err := goredis.Bytes(conn.Do("LOCK", v...))
	if err != nil {
		conn.Close()
		return err
	}
	l.id = id
	l.conn = conn
	return nil
}

func (l *respLocker) Unlock() error {
	if l.id == nil {
		return fmt.Errorf("no lock id")
	}

	_, err := l.conn.Do("UNLOCK", l.id)
	l.conn.Close()
	if err != nil {
		l.id = nil
	}

	return err
}

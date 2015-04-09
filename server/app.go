package server

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/siddontang/tlock/lock"
)

type App struct {
	m sync.Mutex

	wg sync.WaitGroup

	httpListener net.Listener

	keyLockerGroup  *lock.KeyLockerGroup
	pathLockerGroup *lock.PathLockerGroup
}

func NewApp() *App {
	a := new(App)

	a.keyLockerGroup = lock.NewKeyLockerGroup()
	a.pathLockerGroup = lock.NewPathLockerGroup()

	return a
}

func (a *App) StartHTTP(addr string) error {
	a.m.Lock()
	defer a.m.Unlock()

	var err error
	a.httpListener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()

		mux := http.NewServeMux()
		mux.Handle("/lock", a.newLockHandler())

		http.Serve(a.httpListener, mux)

	}()
	return nil
}

func (a *App) Close() {
	a.m.Lock()
	defer a.m.Unlock()

	if a.httpListener != nil {
		a.httpListener.Close()
	}

	a.wg.Wait()
}

func (a *App) Lock(tp string, keys []string) error {
	b, err := a.LockTimeout(tp, lock.InfiniteTimeout, keys)
	if !b {
		panic("Wait lock too long, panic")
	}
	return err
}

func (a *App) LockTimeout(tp string, timeout time.Duration, keys []string) (bool, error) {
	if len(keys) == 0 {
		return false, fmt.Errorf("empty lock keys")
	}

	switch strings.ToLower(tp) {
	case "key":
		return a.keyLockerGroup.LockTimeout(timeout, keys...), nil
	case "path":
		return a.pathLockerGroup.LockTimeout(timeout, keys...), nil
	default:
		return false, fmt.Errorf("invalid lock type %s", tp)
	}
}

func (a *App) Unlock(tp string, keys []string) error {
	if len(keys) == 0 {
		return fmt.Errorf("empty lock keys")
	}

	switch strings.ToLower(tp) {
	case "key":
		a.keyLockerGroup.Unlock(keys...)
	case "path":
		a.pathLockerGroup.Unlock(keys...)
	default:
		return fmt.Errorf("invalid lock type %s", tp)
	}

	return nil
}

type lockHandler struct {
	a *App
}

func (a *App) newLockHandler() *lockHandler {
	h := new(lockHandler)
	h.a = a

	return h
}

// Lock:   Post/Put /lock?keys=a,b,c&timeout=10&type=key
// Unlock: Delete   /lock?keys=a,b,c
// For HTTP, the default and maximum timeout is 60s
// Lock type supports key and path, the default is key
func (h *lockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST", "PUT":
		keys := strings.Split(r.FormValue("keys"), ",")
		timeout, _ := strconv.Atoi(r.FormValue("timeout"))
		if timeout <= 0 || timeout > 60 {
			timeout = 60
		}
		tp := r.FormValue("type")
		if len(tp) == 0 {
			tp = "key"
		}

		b, err := h.a.LockTimeout(tp, time.Duration(timeout)*time.Second, keys)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		} else if !b {
			w.WriteHeader(http.StatusRequestTimeout)
			w.Write([]byte("Lock timeout"))
		} else {
			w.WriteHeader(http.StatusOK)
		}
	case "DELETE":
		keys := strings.Split(r.FormValue("keys"), ",")
		tp := r.FormValue("type")
		if len(tp) == 0 {
			tp = "key"
		}

		err := h.a.Unlock(tp, keys)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

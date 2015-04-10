package server

import (
	"bytes"
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

	locksMutex sync.Mutex
	keyLocks   map[string]time.Time
	pathLocks  map[string]time.Time
}

func NewApp() *App {
	a := new(App)

	a.keyLockerGroup = lock.NewKeyLockerGroup()
	a.pathLockerGroup = lock.NewPathLockerGroup()

	a.keyLocks = make(map[string]time.Time, 1024)
	a.pathLocks = make(map[string]time.Time, 1024)

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

func (a *App) HTTPAddr() net.Addr {
	if a.httpListener == nil {
		return nil
	} else {
		return a.httpListener.Addr()
	}
}

func (a *App) Lock(tp string, names []string) error {
	b, err := a.LockTimeout(tp, lock.InfiniteTimeout, names)
	if !b {
		panic("Wait lock too long, panic")
	}
	return err
}

func (a *App) LockTimeout(tp string, timeout time.Duration, names []string) (bool, error) {
	if len(names) == 0 {
		return false, fmt.Errorf("empty lock names")
	}

	switch strings.ToLower(tp) {
	case "key":
		return a.keyLockerGroup.LockTimeout(timeout, names...), nil
	case "path":
		return a.pathLockerGroup.LockTimeout(timeout, names...), nil
	default:
		return false, fmt.Errorf("invalid lock type %s", tp)
	}
}

func (a *App) Unlock(tp string, names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("empty lock names")
	}

	switch strings.ToLower(tp) {
	case "key":
		a.keyLockerGroup.Unlock(names...)
	case "path":
		a.pathLockerGroup.Unlock(names...)
	default:
		return fmt.Errorf("invalid lock type %s", tp)
	}

	return nil
}

func (a *App) addLockNames(names string, tp string) {
	a.locksMutex.Lock()
	if tp == "key" {
		a.keyLocks[names] = time.Now()
	} else {
		a.pathLocks[names] = time.Now()
	}
	a.locksMutex.Unlock()
}

func (a *App) delLockNames(names string, tp string) {
	a.locksMutex.Lock()
	if tp == "key" {
		delete(a.keyLocks, names)
	} else {
		delete(a.pathLocks, names)
	}
	a.locksMutex.Unlock()
}

const timeFormat string = "2006-01-02 15:04:05"

func (a *App) dumpLockNames() []byte {
	var buf bytes.Buffer

	a.locksMutex.Lock()
	defer a.locksMutex.Unlock()

	buf.WriteString("key lock:\n")
	for names, t := range a.keyLocks {
		buf.WriteString(fmt.Sprintf("%s\t%s\n", names, t.Format(timeFormat)))
	}

	buf.WriteString("\npath lock:\n")
	for names, t := range a.pathLocks {
		buf.WriteString(fmt.Sprintf("%s\t%s\n", names, t.Format(timeFormat)))
	}

	return buf.Bytes()
}

type lockHandler struct {
	a *App
}

func (a *App) newLockHandler() *lockHandler {
	h := new(lockHandler)
	h.a = a

	return h
}

// Lock:   Post/Put /lock?names=a,b,c&timeout=10&type=key
// Unlock: Delete   /lock?names=a,b,c
// For HTTP, the default and maximum timeout is 60s
// Lock type supports key and path, the default is key
// List locks: Get  /lock
func (h *lockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		buf := h.a.dumpLockNames()
		w.Header().Set("Content-Type", "text/plain")
		w.Write(buf)
		return
	case "POST", "PUT":
		names := strings.Split(r.FormValue("names"), ",")
		if len(names) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("empty lock names"))
			return
		}

		timeout, _ := strconv.Atoi(r.FormValue("timeout"))

		if timeout <= 0 || timeout > 60 {
			timeout = 60
		}
		tp := strings.ToLower(r.FormValue("type"))
		if len(tp) == 0 {
			tp = "key"
		}

		b, err := h.a.LockTimeout(tp, time.Duration(timeout)*time.Second, names)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		} else if !b {
			w.WriteHeader(http.StatusRequestTimeout)
			w.Write([]byte("Lock timeout"))
		} else {
			h.a.addLockNames(r.FormValue("names"), tp)
			w.WriteHeader(http.StatusOK)
		}
	case "DELETE":
		names := strings.Split(r.FormValue("names"), ",")
		if len(names) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("empty lock names"))
			return
		}

		tp := strings.ToLower(r.FormValue("type"))
		if len(tp) == 0 {
			tp = "key"
		}

		err := h.a.Unlock(tp, names)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		} else {
			h.a.delLockNames(r.FormValue("names"), tp)
			w.WriteHeader(http.StatusOK)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

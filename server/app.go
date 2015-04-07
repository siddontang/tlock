package server

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type App struct {
	m sync.Mutex

	wg sync.WaitGroup

	httpListener net.Listener
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

func (a *App) Lock(tp string, keys []string) {

}

func (a *App) LockTimeout(tp string, timeout time.Duration, keys []string) {

}

func (a *App) Unlock(tp string, keys []string) {

}

type lockHandler struct {
	a *App
}

func (a *App) newLockHandler() *lockHandler {
	h := new(lockHandler)
	h.a = a

	return h
}

// Lock:   Post/Put /lock?keys=a,b,c
// Unlock: Delete   /lock?keys=a,b,c
func (h *lockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST", "PUT":
	case "DELETE":
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

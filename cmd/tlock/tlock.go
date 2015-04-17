package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/siddontang/tlock"
)

var addr = flag.String("addr", "127.0.0.1:13000", "http listen address")
var httpAddr = flag.String("http_addr", "", "http listen address")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	a := tlock.NewApp()

	a.StartRESP(*addr)

	if len(*httpAddr) > 0 {
		a.StartHTTP(*httpAddr)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	<-sc

	a.Close()
}

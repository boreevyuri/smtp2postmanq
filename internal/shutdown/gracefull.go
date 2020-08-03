package shutdown

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type IGracefullShutdown interface {
	IsShutdown() bool
	ShutdownSuccess()
}

type gracefullShudtown struct {
	shutdownChan chan bool
	sig          string
	wg           *sync.WaitGroup
}

func Provide() IGracefullShutdown {
	gs := &gracefullShudtown{shutdownChan: make(chan bool, 1), wg: new(sync.WaitGroup)}
	gs.wg.Add(1)
	go gs.initSignal()
	go gs.shutdown()
	return gs
}

func (gs *gracefullShudtown) initSignal() {
	var gracefulStop = make(chan os.Signal)

	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	for s := range gracefulStop {
		gs.sig = s.String()
		gs.wg.Done()
		close(gs.shutdownChan)
		return
	}
}

func (gs *gracefullShudtown) shutdown() {
	gs.wg.Wait()
	os.Exit(0)
}

func (gs *gracefullShudtown) ShutdownSuccess() {
	gs.wg.Done()
}

func (gs *gracefullShudtown) IsShutdown() bool {
	gs.wg.Add(1)
	<-gs.shutdownChan
	return true
}

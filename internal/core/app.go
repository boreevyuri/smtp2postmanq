package core

import (
	"context"
	"os"
	"os/signal"

	"golang.org/x/sync/errgroup"
)

// Application - container of runnable objects with run logic and graceful stop
type Application struct {
	runnables []Runnable
	cancel    context.CancelFunc
}

func NewApplication() *Application {
	return &Application{}
}

type Runnable interface {
	Run(ctx context.Context) error
}

func (a *Application) Register(r Runnable) {
	a.runnables = append(a.runnables, r)
}

func (a *Application) Run(ctx context.Context) error {
	ctx, a.cancel = context.WithCancel(ctx)
	eg, ctx := errgroup.WithContext(ctx)
	for _, r := range a.runnables {
		eg.Go(newRunFn(ctx, r))
	}
	go a.waitForInterruption()
	return eg.Wait()
}

func (a *Application) stop() {
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *Application) waitForInterruption() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	a.stop()
}

func newRunFn(ctx context.Context, r Runnable) func() error {
	return func() error {
		return r.Run(ctx)
	}
}

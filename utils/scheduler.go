package utils

import (
	"sync"
	"time"
)

type interval struct {
	sync.Mutex
	ticker *time.Ticker
}

func (i *interval) Stop() {
	i.ticker.Stop()
}

func (i *interval) WrapInLock(f func()) {
	i.Lock()
	f()
	i.Unlock()
}

// Create an interval that run f function after every amount of time (duration)
func CreateInterval(f func(), duration time.Duration) *interval {
	i := &interval{ticker: time.NewTicker(duration)}
	go func() {
		for {
			i.Lock()
			f()
			i.Unlock()
			<-i.ticker.C
		}
	}()
	return i
}

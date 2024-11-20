package bwlimit

import (
	"sync"
	"time"
)

type OperationTicker struct {
	mu sync.Mutex
	ch chan struct{}
	fn func()
}

var Ticker *OperationTicker

func (ot *OperationTicker) SetOnTick(fn func()) {
	ot.mu.Lock()
	ot.fn = fn
	ot.mu.Unlock()
}

func (ot *OperationTicker) GetOnTick() (fn func()) {
	ot.mu.Lock()
	fn = ot.fn
	ot.mu.Unlock()
	return
}

func (ot *OperationTicker) TickCh() (ch <-chan struct{}) {
	ot.mu.Lock()
	ch = ot.ch
	ot.mu.Unlock()
	return
}

func (ot *OperationTicker) run() {
	defer close(ot.ch)

	tckr := time.NewTicker(interval)
	defer tckr.Stop()

	for range tckr.C {
		ot.mu.Lock()
		oldCh := ot.ch
		ot.ch = make(chan struct{})
		ot.mu.Unlock()
		close(oldCh)
	}
}

func (ot *OperationTicker) runOnTick() {
	for {
		<-ot.TickCh()
		if fn := ot.GetOnTick(); fn != nil {
			fn()
		}
	}
}

func init() {
	Ticker = &OperationTicker{
		ch: make(chan struct{}),
	}
	go Ticker.run()
	go Ticker.runOnTick()
}

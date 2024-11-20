package bwlimit

import (
	"sync"
	"time"
)

// A Ticker synchronizes rate calculation among multiple Limiters and
// provides the on-tick callback.
type Ticker struct {
	mu sync.Mutex
	ch chan struct{}
	fn func()
}

var DefaultTicker *Ticker

// NewLimiter returns a new Limiter using this Ticker.
// If you provide limits, the first will set
// both read and write limits, the second will set the write limit.
//
// To stop the limiter and free it's resources, call Stop.
func (ot *Ticker) NewLimiter(limits ...int64) *Limiter {
	return &Limiter{
		Ticker: ot,
		Reads:  NewOperation(limits, 0),
		Writes: NewOperation(limits, 1),
	}
}

func (ot *Ticker) SetOnTick(fn func()) {
	ot.mu.Lock()
	ot.fn = fn
	ot.mu.Unlock()
}

func (ot *Ticker) GetOnTick() (fn func()) {
	ot.mu.Lock()
	fn = ot.fn
	ot.mu.Unlock()
	return
}

func (ot *Ticker) Ch() (ch <-chan struct{}) {
	ot.mu.Lock()
	ch = ot.ch
	ot.mu.Unlock()
	return
}

func (ot *Ticker) run() {
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

func (ot *Ticker) runOnTick() {
	for {
		<-ot.Ch()
		if fn := ot.GetOnTick(); fn != nil {
			fn()
		}
	}
}

func init() {
	DefaultTicker = &Ticker{
		ch: make(chan struct{}),
	}
	go DefaultTicker.run()
	go DefaultTicker.runOnTick()
}

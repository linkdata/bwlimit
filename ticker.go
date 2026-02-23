package bwlimit

import (
	"sync"
	"time"
)

// A Ticker synchronizes rate calculation among multiple Limiters.
// Ticker values must be created with NewTicker; the zero value is not supported.
type Ticker struct {
	mu sync.Mutex
	ch chan struct{}
}

var DefaultTicker *Ticker = NewTicker()

// NewTicker creates and starts a Ticker.
func NewTicker() (ot *Ticker) {
	ot = &Ticker{ch: make(chan struct{})}
	go ot.run()
	return
}

// NewLimiter returns a new Limiter using this Ticker.
// If you provide limits, the first will set
// both read and write limits, the second will set the write limit.
// Limits are applied in 100ms slices with fractional carry-over between
// slices, so very low rates are accurate over time but can be bursty
// at slice boundaries.
//
// To stop the limiter and free it's resources, call Stop.
func (ot *Ticker) NewLimiter(limits ...int64) (l *Limiter) {
	return &Limiter{
		Ticker: ot,
		Reads:  NewOperation(ot, limits, 0),
		Writes: NewOperation(ot, limits, 1),
	}
}

// WaitCh returns a channel that will close when the current rate limit
// time slice runs out.
func (ot *Ticker) WaitCh() (ch <-chan struct{}) {
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

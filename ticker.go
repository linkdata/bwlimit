package bwlimit

import (
	"sync"
	"time"
)

// A Ticker synchronizes rate calculation among multiple Limiters.
// Ticker values must be created with NewTicker; the zero value is not supported.
type Ticker struct {
	mu     sync.Mutex
	ch     chan struct{}
	stopCh chan struct{}
	doneCh chan struct{}
}

var DefaultTicker *Ticker = NewTicker()

// NewTicker creates and starts a Ticker.
func NewTicker() (ot *Ticker) {
	ot = &Ticker{
		ch:     make(chan struct{}),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	go ot.run(ot.stopCh)
	return
}

// Stop stops the Ticker and closes the current WaitCh channel.
func (ot *Ticker) Stop() {
	ot.mu.Lock()
	if ch := ot.stopCh; ch != nil {
		ot.stopCh = nil
		close(ch)
	}
	ot.mu.Unlock()
	<-ot.doneCh
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

func (ot *Ticker) run(stopCh chan struct{}) {
	defer func() {
		close(ot.ch)
		close(ot.doneCh)
	}()

	tckr := time.NewTicker(interval)
	defer tckr.Stop()

	for {
		select {
		case <-tckr.C:
			newCh := make(chan struct{})
			ot.mu.Lock()
			oldCh := ot.ch
			ot.ch = newCh
			ot.mu.Unlock()
			close(oldCh)
		case <-stopCh:
			return
		}
	}
}

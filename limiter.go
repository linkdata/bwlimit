package bwlimit

import (
	"net"
)

var DefaultNetDialer = &net.Dialer{}

type Limiter struct {
	*Ticker
	Reads  *Operation
	Writes *Operation
}

// NewLimiter returns a new limiter from DefaultTicker.
// If you provide limits, the first will set
// both read and write limits, the second will set the write limit.
//
// To stop the Limiter and free it's resources, call Stop.
func NewLimiter(limits ...int64) *Limiter {
	return DefaultTicker.NewLimiter(limits...)
}

// Stop stops the Limiter and frees any resources. Reads and writes on
// a stopped and rate-limited Limiter returns io.EOF. On an unlimited
// Limiter they function as normal.
func (l *Limiter) Stop() {
	l.Reads.Stop()
	l.Writes.Stop()
}

// alreadyLimits returns true if cd is already limited by this Limiter.
// This lets us help the user avoiding double-accounting bandwidth.
func (l *Limiter) alreadyLimits(cd ContextDialer) bool {
	for {
		if d, ok := cd.(*Dialer); ok {
			if d.Limiter == l {
				return true
			}
			cd = d.ContextDialer
		} else {
			return false
		}
	}
}

// Wrap returns a ContextDialer wrapping cd that is bandwidth limited by this Limiter.
//
// If cd is nil we use DefaultNetDialer. If cd is already limited by this Limiter, cd
// is returned unchanged.
func (l *Limiter) Wrap(cd ContextDialer) ContextDialer {
	if cd == nil {
		cd = DefaultNetDialer
	}
	if l.alreadyLimits(cd) {
		return cd
	}
	return &Dialer{ContextDialer: cd, Limiter: l}
}

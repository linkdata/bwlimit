package bwlimit

import (
	"context"
	"net"
)

type DialFn func(network string, address string) (net.Conn, error)
type DialContextFn func(ctx context.Context, network string, address string) (net.Conn, error)

var DefaultNetDialer = &net.Dialer{}

type Limiter struct {
	Reads  *Operation
	Writes *Operation
}

// NewLimiter returns a new limiter. If you provide limits, the first will set
// both read and write limits, the second will set the write limit.
func NewLimiter(limits ...int64) *Limiter {
	return &Limiter{
		Reads:  NewOperation(limits, 0),
		Writes: NewOperation(limits, 1),
	}
}

func (l *Limiter) Stop() {
	l.Reads.Stop()
	l.Writes.Stop()
}

// Wrap returns a DialContextFn using the given fn that is bandwidth limited by this Limiter.
// If fn is nil we use DefaultNetDialer.DialContext.
func (l *Limiter) Wrap(fn DialContextFn) DialContextFn {
	if fn == nil {
		fn = DefaultNetDialer.DialContext
	}
	d := &Dialer{DialContextFn: fn, Limiter: l}
	return d.DialContext
}

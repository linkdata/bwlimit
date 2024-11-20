package bwlimit

import (
	"context"
	"net"
)

type DialContextFn func(ctx context.Context, network string, address string) (net.Conn, error)

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

// Wrap returns a DialContextFn using the given fn that is bandwidth limited by this Limiter.
// If fn is nil we use DefaultNetDialer.DialContext.
func (l *Limiter) Wrap(fn DialContextFn) DialContextFn {
	if fn == nil {
		fn = DefaultNetDialer.DialContext
	}
	d := &Dialer{DialContextFn: fn, Limiter: l}
	return d.DialContext
}

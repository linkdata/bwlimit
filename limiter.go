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

func NewLimiter(ctx context.Context) *Limiter {
	return &Limiter{
		Reads:  NewOperation(ctx, true),
		Writes: NewOperation(ctx, false),
	}
}

// Wrap returns a DialContextFn using the given fn that is bandwidth limited by this Limiter.
// If fn is nil we use DefaultNetDialer.DialContext.
func (l *Limiter) Wrap(fn DialContextFn) DialContextFn {
	if fn == nil {
		fn = DefaultNetDialer.DialContext
	}
	return Dialer{Limiter: l, DialContextFn: fn}.DialContextFn
}

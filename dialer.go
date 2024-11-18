package bwlimit

import (
	"context"
	"net"
)

type DialContextFn func(ctx context.Context, network string, address string) (net.Conn, error)

type Dialer struct {
	DialContextFn // DialContext function we wrap, if nil use net.Dialer.DialContext
	*Limiter      // Limiter to use
}

var defaultNetDialer = net.Dialer{}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (conn net.Conn, err error) {
	fn := d.DialContextFn
	if fn == nil {
		fn = defaultNetDialer.DialContext
	}
	if conn, err = fn(ctx, network, address); err == nil {
		conn = &Conn{
			Conn:    conn,
			Limiter: d.Limiter,
		}
	}
	return
}

func (d *Dialer) Dial(network string, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

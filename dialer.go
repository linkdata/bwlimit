package bwlimit

import (
	"context"
	"net"
)

type Dialer struct {
	DialContextFn // DialContext function we wrap
	*Limiter      // Limiter to use
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (conn net.Conn, err error) {
	if conn, err = d.DialContextFn(ctx, network, address); err == nil {
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

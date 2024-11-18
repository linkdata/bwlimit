package bwlimit

import "net"

type Listener struct {
	net.Listener // underlying net.Listener
	*Limiter     // Limiter to use
}

func (l *Listener) Accept() (conn net.Conn, err error) {
	if conn, err = l.Listener.Accept(); err == nil {
		conn = &Conn{
			Conn:    conn,
			Limiter: l.Limiter,
		}
	}
	return
}

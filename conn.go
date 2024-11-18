package bwlimit

import (
	"net"
)

type Conn struct {
	net.Conn // underlying net.Conn
	*Limiter // Limiter to use
}

func (c *Conn) Read(b []byte) (n int, err error) {
	return c.Limiter.Reads.io(c.Conn.Read, b)
}

func (c *Conn) Write(b []byte) (n int, err error) {
	return c.Limiter.Writes.io(c.Conn.Write, b)
}

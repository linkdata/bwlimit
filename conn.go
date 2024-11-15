package limitedconn

import (
	"net"
	"sync/atomic"
	"time"
)

type Conn struct {
	net.Conn // underlying net.Conn
	when     time.Time
	counters []atomic.Uint64
}

func (c *Conn) limit() (n int) {
	return
}

func (c *Conn) Read(b []byte) (n int, err error) {

	return c.Conn.Read(b)
}

func (c *Conn) Write(b []byte) (n int, err error) {
	return c.Conn.Write(b)
}

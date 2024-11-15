package bwlimit

import (
	"net"
)

type Conn struct {
	*Dialer  // Dialer we belong to
	net.Conn // underlying net.Conn
}

func (c *Conn) Read(b []byte) (n int, err error) {
	c.Dialer.readers.Add(1)
	defer c.Dialer.readers.Add(^uint32(0))
	for len(b) > 0 && err == nil {
		var done int
		todo := int(max(1, c.Dialer.readAvail.Load()/c.Dialer.readers.Load()))
		if done, err = c.Conn.Read(b[:todo]); done > 0 {
			n += done
			b = b[done:]
			if done < todo {
				break
			}
		}
	}
	return
}

func (c *Conn) Write(b []byte) (n int, err error) {
	c.Dialer.writers.Add(1)
	defer c.Dialer.writers.Add(^uint32(0))
	for len(b) > 0 && err == nil {
		var done int
		todo := int(max(1, c.Dialer.writeAvail.Load()/c.Dialer.writers.Load()))
		if done, err = c.Conn.Write(b[:todo]); done > 0 {
			n += done
			b = b[done:]
		}
	}
	return
}

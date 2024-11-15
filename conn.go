package bwlimit

import (
	"context"
	"io"
	"net"
	"sync/atomic"
)

type Conn struct {
	*Dialer  // Dialer we belong to
	net.Conn // underlying net.Conn
	ctx      context.Context
}

func (c *Conn) Read(b []byte) (n int, err error) {
	for len(b) > 0 && err == nil {
		select {
		case _, ok := <-c.Dialer.readCh:
			if !ok {
				err = io.ErrUnexpectedEOF
				return
			}
		case <-c.ctx.Done():
			err = c.ctx.Err()
			return
		}
		var done int
		todo := min(len(b), 1024)
		done, err = c.Conn.Read(b[:todo])
		atomic.AddInt32(&c.Dialer.readAvail, int32(1024-done))
		if done > 0 {
			atomic.AddInt32(&c.Dialer.readCount, int32(done))
			n += done
			b = b[done:]
		}
		if done < todo {
			break
		}
	}
	return
}

func (c *Conn) Write(b []byte) (n int, err error) {
	for len(b) > 0 && err == nil {
		select {
		case _, ok := <-c.Dialer.writeCh:
			if !ok {
				err = io.ErrUnexpectedEOF
				return
			}
		case <-c.ctx.Done():
			err = c.ctx.Err()
			return
		}
		var done int
		todo := min(len(b), 1024)
		done, err = c.Conn.Write(b[:todo])
		atomic.AddInt32(&c.Dialer.writeAvail, int32(1024-done))
		if done > 0 {
			atomic.AddInt32(&c.Dialer.writeCount, int32(done))
			n += done
			b = b[done:]
		}
	}
	return
}

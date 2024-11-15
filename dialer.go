package bwlimit

import (
	"context"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Dialer struct {
	net.Dialer // underlying net.Dialer
	ReadLimit  atomic.Uint32
	WriteLimit atomic.Uint32
	readers    atomic.Uint32
	writers    atomic.Uint32
	readAvail  atomic.Uint32
	writeAvail atomic.Uint32
	init       sync.Once
}

func (d *Dialer) ticker() {
	const secparts = 10
	const interval = time.Second / secparts
	now := time.Now()
	toSleep := interval
	for {
		time.Sleep(toSleep)
		if elapsed := time.Since(now); elapsed > 0 {
			now.Add(elapsed)
			toSleep += interval - elapsed
			if rl := d.ReadLimit.Load(); rl > 0 {
				if d.readAvail.Add(max(1, rl/secparts)) > rl {
					d.readAvail.Store(rl)
				}
			} else {
				d.readAvail.Store(math.MaxUint32)
			}
			if wl := d.WriteLimit.Load(); wl > 0 {
				if d.writeAvail.Add(max(1, wl/secparts)) > wl {
					d.writeAvail.Store(wl)
				}
			} else {
				d.writeAvail.Store(math.MaxUint32)
			}
		}
	}
}

func (d *Dialer) initialize() {
	go d.ticker()
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (conn net.Conn, err error) {
	d.init.Do(d.initialize)
	if conn, err = d.Dialer.DialContext(ctx, network, address); err == nil {
		conn = &Conn{
			Dialer: d,
			Conn:   conn,
		}
	}
	return
}

func (d *Dialer) Dial(network string, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

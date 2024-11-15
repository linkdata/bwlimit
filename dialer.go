package bwlimit

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Dialer struct {
	net.Dialer // underlying net.Dialer
	ReadLimit  int32
	WriteLimit int32
	ReadRate   int32
	WriteRate  int32
	readAvail  int32
	writeAvail int32
	readCount  int32
	writeCount int32
	init       sync.Once
	readCh     chan struct{}
	writeCh    chan struct{}
}

const secparts = 100
const interval = time.Second / secparts

func limiter(ch chan struct{}, limit, avail, count, rate *int32) {
	defer close(ch)

	now := time.Now()
	toSleep := interval
	seccount := 0
	var accum int32
	for {
		time.Sleep(toSleep)
		if elapsed := time.Since(now); elapsed > 0 {
			seccount++
			now = now.Add(elapsed)
			toSleep += interval - elapsed
			accum += atomic.LoadInt32(limit) / secparts
			accum += atomic.SwapInt32(avail, 0)
			for accum >= 1024 {
				accum -= 1024
				select {
				case ch <- struct{}{}:
				default:
				}
			}
			if seccount%secparts == 0 {
				atomic.StoreInt32(rate, atomic.SwapInt32(count, 0))
			}
		}
	}
}

func (d *Dialer) initialize() {
	d.readCh = make(chan struct{}, secparts)
	d.writeCh = make(chan struct{}, secparts)
	go limiter(d.readCh, &d.ReadLimit, &d.readAvail, &d.readCount, &d.ReadRate)
	go limiter(d.writeCh, &d.WriteLimit, &d.writeAvail, &d.writeCount, &d.WriteRate)
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (conn net.Conn, err error) {
	d.init.Do(d.initialize)
	if conn, err = d.Dialer.DialContext(ctx, network, address); err == nil {
		conn = &Conn{
			Dialer: d,
			Conn:   conn,
			ctx:    ctx,
		}
	}
	return
}

func (d *Dialer) Dial(network string, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

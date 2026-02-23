package bwlimit

import (
	"io"
	"sync"
	"sync/atomic"
	"time"
)

const secparts = 10
const interval = time.Second / secparts
const batchsize = 4096

type Operation struct {
	*Ticker              // Ticker we belong to
	Limit   atomic.Int64 // bandwith limit in bytes/sec
	Rate    atomic.Int64 // current rate in bytes/sec
	Count   atomic.Int64 // number of bytes seen
	avail   atomic.Int64
	count   atomic.Int64
	ch      <-chan int64
	doneCh  chan struct{}
	reader  bool
	mu      sync.Mutex // protects following
	stopCh  chan struct{}
}

func NewOperation(t *Ticker, limits []int64, idx int) (op *Operation) {
	ch := make(chan int64)
	op = &Operation{
		Ticker: t,
		ch:     ch,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
		reader: idx == 0,
	}
	var limit int64
	if len(limits) > 0 {
		limit = limits[0]
		if len(limits) > idx {
			limit = limits[idx]
		}
	}
	op.Limit.Store(limit)
	go op.run(ch)
	return
}

func (op *Operation) Stop() {
	op.mu.Lock()
	ch := op.stopCh
	op.stopCh = nil
	op.mu.Unlock()
	if ch != nil {
		close(ch)
	}
	<-op.doneCh
}

func (op *Operation) run(ch chan<- int64) {
	defer func() {
		close(ch)
		op.Count.Add(op.count.Swap(0))
		close(op.doneCh)
	}()

	op.mu.Lock()
	stopCh := op.stopCh
	seccount := 0
	counts := make([]int64, secparts)
	carry := int64(0)
	op.mu.Unlock()

	if stopCh != nil {
		for {
			var limitCh chan<- int64
			var todo int64
			var batch int64
			if limit := op.Limit.Load(); limit > 0 {
				carry += limit
				todo = carry / secparts
				carry = carry % secparts
				todo += op.avail.Swap(0)
				if todo > 0 {
					limitCh = ch
					batch = min(batchsize, todo)
				}
			} else {
				carry = 0
			}
			waitCh := op.WaitCh()

		partialsecond:
			for {
				select {
				case <-stopCh:
					return
				case <-op.Ticker.doneCh:
					return
				case limitCh <- batch:
					todo -= batch
					todo += op.avail.Swap(0)
					if todo < batch {
						<-waitCh
						break partialsecond
					}
				case <-waitCh:
					break partialsecond
				}
			}

			count := op.count.Swap(0)
			op.Count.Add(count)
			counts[seccount] = count
			seccount++
			if seccount >= secparts {
				seccount = 0
			}
			var rate int64
			for i := 0; i < secparts; i++ {
				rate += counts[i]
			}
			op.Rate.Store(rate)
		}
	}
}

func (op *Operation) io(fn func([]byte) (int, error), b []byte) (n int, err error) {
outer:
	for len(b) > 0 && err == nil {
		var done int
		if op.Limit.Load() < 1 {
			done, err = fn(b)
			n += done
			op.count.Add(int64(done))
			return
		}
		select {
		case batch, ok := <-op.ch:
			err = io.EOF
			if ok {
				todo := min(int64(len(b)), batch)
				done, err = fn(b[:todo])
				op.avail.Add(batch - int64(done))
				if done > 0 {
					op.count.Add(int64(done))
					n += int(done)
					b = b[done:]
				}
				if op.reader && int64(done) < todo {
					break outer
				}
			}
		case <-op.WaitCh():
		}
	}

	if op.reader && n > 0 && err == io.EOF {
		err = nil
	}
	return
}

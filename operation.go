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
	Limit  atomic.Int64 // bandwith limit in bytes/sec
	Rate   atomic.Int64 // current rate in bytes/sec
	Count  atomic.Int64 // number of bytes seen
	avail  atomic.Int64
	count  atomic.Int64
	ch     <-chan int
	reader bool
	mu     sync.Mutex // protects following
	stopCh chan struct{}
}

func NewOperation(limits []int64, idx int) (op *Operation) {
	ch := make(chan int)
	op = &Operation{
		ch:     ch,
		stopCh: make(chan struct{}),
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
}

func (op *Operation) run(ch chan<- int) {
	defer close(ch)

	op.mu.Lock()
	stopCh := op.stopCh
	seccount := 0
	counts := make([]int64, secparts)
	op.mu.Unlock()

	if stopCh != nil {
		for {
			var limitCh chan<- int
			var todo int
			var batch int
			if limit := op.Limit.Load(); limit > 0 {
				limitCh = ch
				todo = max(1, int(limit/secparts))
				batch = min(batchsize, todo)
			}
			tickCh := Ticker.TickCh()

		partialsecond:
			for {
				select {
				case <-stopCh:
					return
				case limitCh <- batch:
					todo -= batch
					todo += int(op.avail.Swap(0))
					if todo < batch {
						<-tickCh
						break partialsecond
					}
				case <-tickCh:
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
	if op.Limit.Load() < 1 {
		n, err = fn(b)
		op.count.Add(int64(n))
		return
	}
	for len(b) > 0 && err == nil {
		batch, ok := <-op.ch
		err = io.EOF
		if ok {
			var done int
			todo := min(len(b), batch)
			done, err = fn(b[:todo])
			op.avail.Add(int64(batch - done))
			if done > 0 {
				op.count.Add(int64(done))
				n += done
				b = b[done:]
			}
			if op.reader && done < todo {
				break
			}
		}
	}
	if op.reader && n > 0 && err == io.EOF {
		err = nil
	}
	return
}

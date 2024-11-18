package bwlimit

import (
	"context"
	"io"
	"sync/atomic"
	"time"
)

const secparts = 10
const interval = time.Second / secparts

type Operation struct {
	Limit  atomic.Int32 // bandwith limit in bytes/sec
	Rate   atomic.Int32 // current rate in bytes/sec
	avail  atomic.Int32
	count  atomic.Int32
	batch  atomic.Int32
	ch     <-chan struct{}
	reader bool
}

func NewOperation(ctx context.Context, reader bool) (op *Operation) {
	ch := make(chan struct{}, secparts)
	op = &Operation{ch: ch, reader: reader}
	go op.run(ctx, ch)
	return
}

func (op *Operation) run(ctx context.Context, ch chan<- struct{}) {
	seccount := 0
	now := time.Now()
	defer close(ch)

	for {
		if elapsed := time.Since(now); elapsed > 0 {
			now = now.Add(elapsed)

			if limit := op.Limit.Load(); limit > 0 {
				todo := max(1, limit/secparts)
				batch := min(1024, todo)
				op.batch.Store(batch)
				for todo >= batch && time.Since(now) < (interval-(interval/10)) {
					todo += op.avail.Swap(0)
					select {
					case <-ctx.Done():
						return
					case ch <- struct{}{}:
						todo -= batch
					default:
						time.Sleep(interval / 10)
					}
				}
			}

			time.Sleep(interval - time.Since(now))
			seccount++
			if seccount%secparts == 0 {
				op.Rate.Store(op.count.Swap(0))
			}
		}
	}
}

func (op *Operation) io(fn func([]byte) (int, error), b []byte) (n int, err error) {
	for len(b) > 0 && err == nil {
		if op.Limit.Load() < 1 {
			n, err = fn(b)
			op.count.Add(int32(n)) // #nosec G115
			return
		}
		_, ok := <-op.ch
		err = io.EOF
		if ok {
			var done int
			batch := int(op.batch.Load())
			todo := min(len(b), batch)
			done, err = fn(b[:todo])
			op.avail.Add(int32(batch - done)) // #nosec G115
			if done > 0 {
				op.count.Add(int32(done)) // #nosec G115
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

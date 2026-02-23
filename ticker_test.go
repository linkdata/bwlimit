package bwlimit

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestTicker_NewTicker_NewLimiter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticker := NewTicker()
		defer ticker.Stop()
		l := ticker.NewLimiter(1000)
		defer l.Stop()

		done := make(chan struct{})
		go func() {
			_, _ = l.Reads.io(bytes.NewReader(make([]byte, 1)).Read, make([]byte, 1))
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("read stalled with NewTicker")
		}
	})
}

func TestTicker_Stop_unblocksLimitedOperation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticker := NewTicker()
		l := ticker.NewLimiter(1000)
		defer l.Stop()

		ticker.Stop()

		done := make(chan struct{})
		var n int
		var err error
		go func() {
			n, err = l.Reads.io(bytes.NewReader(make([]byte, 1)).Read, make([]byte, 1))
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("read stalled after ticker stop")
		}

		if n != 0 {
			t.Fatalf("got n=%d, want 0", n)
		}
		if err != io.EOF {
			t.Fatalf("got err=%v, want %v", err, io.EOF)
		}
	})
}

func TestTicker_ConcurrentStopStress(t *testing.T) {
	const (
		iterations = 25
		limiters   = 16
		payload    = 64 * 1024
	)

	for range iterations {
		synctest.Test(t, func(t *testing.T) {
			ticker := NewTicker()
			ls := make([]*Limiter, limiters)
			ioDone := make(chan struct{}, limiters*2)

			for i := range limiters {
				l := ticker.NewLimiter(1000, 1000)
				ls[i] = l

				go func(l *Limiter) {
					_, _ = l.Reads.io(bytes.NewReader(make([]byte, payload)).Read, make([]byte, payload))
					ioDone <- struct{}{}
				}(l)

				go func(l *Limiter) {
					_, _ = l.Writes.io(io.Discard.Write, make([]byte, payload))
					ioDone <- struct{}{}
				}(l)
			}

			// Ensure operations have entered limited mode.
			<-ticker.WaitCh()

			var stopWG sync.WaitGroup
			stopWG.Add(limiters + 1)

			for i, l := range ls {
				go func() {
					defer stopWG.Done()
					if i%2 == 0 {
						<-ticker.WaitCh()
					}
					l.Stop()
				}()
			}

			go func() {
				defer stopWG.Done()
				<-ticker.WaitCh()
				ticker.Stop()
			}()

			stopped := make(chan struct{})
			go func() {
				stopWG.Wait()
				close(stopped)
			}()

			select {
			case <-stopped:
			case <-time.After(10 * time.Second):
				t.Fatal("timeout waiting for concurrent limiter/ticker Stop calls")
			}

			for range limiters * 2 {
				select {
				case <-ioDone:
				case <-time.After(10 * time.Second):
					t.Fatal("timeout waiting for active IO goroutine to return")
				}
			}
		})
	}
}

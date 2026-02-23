package bwlimit

import (
	"bytes"
	"io"
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

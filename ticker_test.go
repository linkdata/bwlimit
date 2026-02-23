package bwlimit

import (
	"bytes"
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

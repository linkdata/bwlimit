package bwlimit

import (
	"bytes"
	"testing"
	"time"
)

func TestTicker_zeroValue_NewLimiter(t *testing.T) {
	l := (&Ticker{}).NewLimiter(1000)
	defer l.Stop()

	done := make(chan struct{})
	go func() {
		_, _ = l.Reads.io(bytes.NewReader(make([]byte, 1)).Read, make([]byte, 1))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("read stalled with zero-value ticker")
	}
}


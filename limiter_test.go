package bwlimit

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestLimiter_Stop(t *testing.T) {
	l := NewLimiter(100000)
	defer l.Stop()

	r := bytes.NewReader(make([]byte, 1000))
	buf := make([]byte, 100)

	n, err := l.Reads.io(r.Read, buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != 100 {
		t.Error(n)
	}
	l.Stop()
	<-l.WaitCh()

	n, err = l.Reads.io(r.Read, buf)
	if err != io.EOF {
		t.Error(err)
	}
	if n != 0 {
		t.Error(n)
	}
}

func TestLimiter_double_Wrap(t *testing.T) {
	l := NewLimiter()
	defer l.Stop()

	d1 := l.Wrap(nil)
	d2 := l.Wrap(d1)
	if d1 != d2 {
		t.Error(d1, d2)
	}
}

func TestLimiter_Stop_flushesCount(t *testing.T) {
	l := NewLimiter(0)

	r := bytes.NewReader(make([]byte, 10))
	buf := make([]byte, 10)
	n, err := l.Reads.io(r.Read, buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != 10 {
		t.Fatal(n)
	}

	l.Stop()

	if got := l.Reads.Count.Load(); got != int64(n) {
		t.Fatalf("got %d want %d", got, n)
	}
}

func TestLimiter_Wrap_cyclicDialerChain_doesNotHang(t *testing.T) {
	l1 := NewLimiter()
	defer l1.Stop()
	l2 := NewLimiter()
	defer l2.Stop()

	d := &Dialer{Limiter: l1}
	d.ContextDialer = d

	done := make(chan ContextDialer, 1)
	go func() {
		done <- l2.Wrap(d)
	}()

	select {
	case wrapped := <-done:
		if wrapped == nil {
			t.Fatal("expected wrapped dialer")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Wrap hung on cyclic dialer chain")
	}
}

func TestLimiter_alreadyLimits_typedNilDialer(t *testing.T) {
	l := NewLimiter()
	defer l.Stop()

	var d *Dialer
	var cd ContextDialer = d
	if l.alreadyLimits(cd) {
		t.Fatal("typed nil dialer should not be detected as already limited")
	}
}

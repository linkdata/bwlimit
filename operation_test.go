package bwlimit

import (
	"bytes"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

type unlimitedReader struct {
	x byte
}

func (ur *unlimitedReader) Read(b []byte) (int, error) {
	x := ur.x
	for i := range b {
		b[i] = x
		x++
	}
	ur.x = x
	return len(b), nil
}

func TestOperation_io_read_nolimit(t *testing.T) {
	const numbytes = (2 * 1024 * 1024 * 1024) - 1

	l := NewLimiter()
	defer l.Stop()
	r := &unlimitedReader{}

	var numread int
	var err error
	var x byte
	buf := make([]byte, 1024*1024)

	now := time.Now()
	for numread < numbytes && err == nil {
		var n int
		n, err = l.Reads.io(r.Read, buf[:(cap(buf)-numread%101)])
		for i := range buf[:n] {
			if buf[i] != x {
				t.Fatal(numread, x)
			}
			x++
			numread++
		}
		if err == nil && time.Since(now) > time.Second {
			break
		}
	}
	elapsed := time.Since(now)
	t.Log("no limit numread", numread, "should be close to", (numread*int(time.Second))/(int(elapsed)))

	if err != nil {
		t.Error(err)
	}
}

func TestOperation_io_read_limit(t *testing.T) {
	l := NewLimiter(100)
	defer l.Stop()

	want := []byte("0123456789")
	r := bytes.NewReader(want)
	got := make([]byte, 100)

	// reading zero bytes returns immediately
	now := time.Now()
	n, err := l.Reads.io(r.Read, got[:0])
	if n != 0 {
		t.Error(n)
	}
	if err != nil {
		t.Error(err)
	}
	if elapsed := time.Since(now); elapsed > interval/2 {
		t.Error("too slow", elapsed)
	}

	n, err = l.Reads.io(r.Read, got)
	if n != len(want) {
		t.Error(n)
	}
	if err != nil {
		t.Error(err)
	}

	got = got[:n]
	if !bytes.Equal(got, want) {
		t.Error(string(got), "!=", string(want))
	}
	if elapsed := time.Since(now); elapsed > interval*2 {
		t.Error(elapsed)
	}
}

func TestOperation_read_rate_low(t *testing.T) {
	l := NewLimiter(1000)
	defer l.Stop()
	r := bytes.NewReader(make([]byte, 2000))
	buf := make([]byte, 1001)

	var tickCount atomic.Int32
	oldOnTick := DefaultTicker.GetOnTick()
	defer DefaultTicker.SetOnTick(oldOnTick)
	<-DefaultTicker.Ch()
	DefaultTicker.SetOnTick(func() { tickCount.Add(1) })

	<-DefaultTicker.Ch()
	// should read in batches of 1000/secparts (=100) bytes
	now := time.Now()
	n, err := l.Reads.io(r.Read, buf)
	elapsed := time.Since(now)
	rate := l.Reads.Rate.Load()
	<-DefaultTicker.Ch()

	if n := tickCount.Load(); n < 11 || n > 13 {
		t.Error(n)
	}
	if n < 990 || n > 1010 {
		t.Error(n)
	}
	if err != nil {
		t.Error(err)
	}

	if elapsed < time.Millisecond*900 || elapsed > time.Millisecond*1100 {
		t.Log(l.Reads.Limit.Load())
		t.Error(elapsed)
	}
	if rate < 990 || rate > 1000 {
		t.Error(rate)
	}
}

func TestOperation_read_rate_high(t *testing.T) {
	const numbytes = (2 * 1024 * 1024 * 1024) - 1
	l := NewLimiter(numbytes)
	defer l.Stop()

	r := &unlimitedReader{}

	var numread int
	var err error
	var x byte
	buf := make([]byte, 1024*1024)

	go func() {
		<-time.NewTimer(time.Second).C
		l.Stop()
	}()

	now := time.Now()
	for numread < numbytes && err == nil {
		var n int
		n, err = l.Reads.io(r.Read, buf[:(cap(buf)-numread%101)])
		for i := range buf[:n] {
			if buf[i] != x {
				t.Fatal(numread, x)
			}
			x++
			numread++
		}
	}
	elapsed := time.Since(now)
	t.Log("high rate numread", numread, "should be close to", (numread*int(time.Second))/(int(elapsed)))

	if err == nil {
		if elapsed < time.Millisecond*900 || elapsed > time.Millisecond*1200 {
			t.Error(elapsed)
		}
		if rate := int(l.Reads.Rate.Load()); rate < numbytes-(numbytes/10) || rate > numbytes {
			t.Error(rate)
		}
	} else if !errors.Is(err, io.EOF) {
		t.Error(err)
	}
}

func TestOperation_write_rate(t *testing.T) {
	l := NewLimiter(1000000)
	defer l.Stop()

	buf := make([]byte, 10000)

	w := bytes.NewBuffer(buf)
	n, err := l.Writes.io(w.Write, []byte("0123456789"))
	if n != 10 {
		t.Error(n)
	}
	if err != nil {
		t.Error(err)
	}

	now := time.Now()
	rate := l.Writes.Rate.Load()
	for rate == 0 && time.Since(now) < time.Second {
		rate = l.Writes.Rate.Load()
	}
	if rate != 10 {
		t.Error(rate)
	}
}

package bwlimit

import (
	"bytes"
	"context"
	"errors"
	"io"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	l := NewLimiter(ctx)

	r := &unlimitedReader{}

	const numbytes = 10*1024*1024 + 1
	var numread int
	var err error
	var x byte
	buf := make([]byte, 1024)
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

	if err != nil {
		t.Error(err)
	}
}

func TestOperation_io_read_limit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	l := NewLimiter(ctx)

	now := time.Now()

	want := []byte("0123456789")
	l.Reads.Limit.Store(100)

	r := bytes.NewReader(want)
	got := make([]byte, 100)
	n, err := l.Reads.io(r.Read, got[:0])
	if n != 0 {
		t.Error(n)
	}
	if err != nil {
		t.Error(err)
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
	if elapsed := time.Since(now); elapsed > interval*3 {
		t.Error(elapsed)
	}
}

func TestOperation_read_rate_low(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	l := NewLimiter(ctx)

	l.Reads.Limit.Store(1000) // 10 bytes @ 1000/sec

	now := time.Now()
	r := bytes.NewReader(make([]byte, 2000))
	buf := make([]byte, 1001)
	n, err := l.Reads.io(r.Read, buf)

	if n < 990 || n > 1010 {
		t.Error(n)
	}
	if err != nil {
		t.Error(err)
	}

	if elapsed := time.Since(now); elapsed < time.Millisecond*900 || elapsed > time.Millisecond*1100 {
		t.Error(elapsed)
	}
	if rate := int(l.Reads.Rate.Load()); rate < 990 || rate > 1000 {
		t.Error(rate)
	}
}

func TestOperation_read_rate_high(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	l := NewLimiter(ctx)

	const numbytes = (2 * 1024 * 1024 * 1024) - 1
	l.Reads.Limit.Store(numbytes)

	now := time.Now()
	r := &unlimitedReader{}

	var numread int
	var err error
	var x byte
	buf := make([]byte, 1024*1024)
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
	t.Log("high rate numread", numread)
	if err == nil {
		if elapsed := time.Since(now); elapsed < time.Millisecond*900 || elapsed > time.Millisecond*1200 {
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	l := NewLimiter(ctx)
	l.Writes.Limit.Store(1000000)

	buf := make([]byte, 10000)

	w := bytes.NewBuffer(buf)
	n, err := l.Writes.io(w.Write, []byte("0123456789"))
	if n != 10 {
		t.Error(n)
	}
	if err != nil {
		t.Error(err)
	}

	rate := l.Writes.Rate.Load()
	for rate == 0 && ctx.Err() == nil {
		rate = l.Writes.Rate.Load()
	}
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
	if rate != 10 {
		t.Error(rate)
	}
}

package bwlimit

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestOperation_io_read_nolimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	l := NewLimiter(ctx)

	want := []byte("some text")

	now := time.Now()
	r := bytes.NewReader(want)
	buf := make([]byte, 100)
	n, err := l.Reads.io(r.Read, buf)
	if n != len(want) {
		t.Error(n)
	}
	if err != nil {
		t.Error(err)
	}
	buf = buf[:n]
	if !bytes.Equal(buf, want) {
		t.Error(string(buf), "!=", string(want))
	}
	if elapsed := time.Since(now); elapsed > interval*2 {
		t.Error(elapsed)
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
	n, err := l.Reads.io(r.Read, got)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	l := NewLimiter(ctx)

	const numbytes = 10 * 1000000
	l.Reads.Limit.Store(numbytes)

	now := time.Now()
	r := bytes.NewReader(make([]byte, numbytes*2))
	var numread int

	for numread < numbytes {
		buf := make([]byte, 1000)
		n, err := l.Reads.io(r.Read, buf)
		numread += n
		if err != nil {
			t.Error(numread, n)
			t.Fatal(err)
		}
	}

	if elapsed := time.Since(now); elapsed < time.Millisecond*900 || elapsed > time.Millisecond*1200 {
		t.Error(elapsed)
	}
	if rate := int(l.Reads.Rate.Load()); rate < numbytes*0.9 || rate > numbytes {
		t.Error(rate)
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

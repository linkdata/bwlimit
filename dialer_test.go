package bwlimit

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDialer_Dial(t *testing.T) {
	l1 := NewLimiter()
	defer l1.Stop()
	l2 := NewLimiter()
	defer l2.Stop()

	d1 := &Dialer{
		Limiter:       l1,
		ContextDialer: l1.Wrap(nil),
	}
	d2 := &Dialer{
		Limiter:       l2,
		ContextDialer: l2.Wrap(d1),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello world!"))
	}))
	defer srv.Close()

	client1 := &http.Client{
		Transport: &http.Transport{
			Dial: d1.Dial,
		},
	}
	client2 := &http.Client{
		Transport: &http.Transport{
			Dial: d2.Dial,
		},
	}
	resp1, err1 := client1.Get(srv.URL)
	resp2, err2 := client2.Get(srv.URL)

	if err1 != nil {
		t.Fatal(err1)
	}
	t.Log(resp1.Status)
	if _, err := io.ReadAll(resp1.Body); err != nil {
		t.Fatal(err)
	}
	if err := resp1.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if err2 != nil {
		t.Fatal(err2)
	}
	t.Log(resp2.Status)
	if _, err := io.ReadAll(resp2.Body); err != nil {
		t.Fatal(err)
	}
	if err := resp2.Body.Close(); err != nil {
		t.Fatal(err)
	}

	r1 := l1.Reads.Count.Load() + l1.Reads.count.Load()
	r2 := l2.Reads.Count.Load() + l2.Reads.count.Load()
	if r1 != r2*2 {
		t.Error(r1, r2)
	}
	if r1 < 1 {
		t.Error(r1)
	}
}

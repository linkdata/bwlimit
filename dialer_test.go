package bwlimit

import (
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
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
	<-l1.WaitCh()
	resp1, err1 := client1.Get(srv.URL)
	resp2, err2 := client2.Get(srv.URL)
	<-l1.WaitCh()

	if err1 != nil {
		t.Fatal(err1)
	}
	t.Log(resp1.Status)

	if err2 != nil {
		t.Fatal(err2)
	}
	t.Log(resp2.Status)

	r1 := l1.Reads.Count.Load()
	r2 := l2.Reads.Count.Load()
	if r1 != r2*2 {
		t.Error(r1, r2)
	}
	if r1 < 1 {
		t.Error(r1)
	}
}

package bwlimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListener_Accept(t *testing.T) {
	l := NewLimiter()
	defer l.Stop()

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	li := &Listener{
		Listener: srv.Listener,
		Limiter:  l,
	}
	srv.Listener = li

	srv.Start()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp.Status)
}

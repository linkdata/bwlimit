package bwlimit_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/linkdata/bwlimit"
)

func ExampleLimiter_NewLimiter() {
	// create a test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello world!"))
	}))
	defer srv.Close()

	// limit reads to 100 bytes/sec, unlimited writes
	lim := bwlimit.NewLimiter(100, 0)
	defer lim.Stop()

	// Clone the default transport so we avoid global side effects.
	tp := http.DefaultTransport.(*http.Transport).Clone()
	tp.DialContext = lim.Wrap(nil).DialContext
	client := &http.Client{Transport: tp}

	// make a request and time it
	now := time.Now()
	resp, err := client.Get(srv.URL)
	elapsed := time.Since(now)

	if err == nil {
		defer resp.Body.Close()
		var body []byte
		if body, err = io.ReadAll(resp.Body); err == nil {
			fmt.Printf("%v %v %q\n", elapsed >= time.Second, lim.Reads.Count.Load() > 100, string(body))
		}
	}
	// Output:
	// true true "Hello world!"
}

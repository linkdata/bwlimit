[![build](https://github.com/linkdata/bwlimit/actions/workflows/go.yml/badge.svg)](https://github.com/linkdata/bwlimit/actions/workflows/go.yml)
[![coverage](https://github.com/linkdata/bwlimit/blob/coverage/main/badge.svg)](https://html-preview.github.io/?url=https://github.com/linkdata/bwlimit/blob/coverage/main/report.html)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/bwlimit)](https://goreportcard.com/report/github.com/linkdata/bwlimit)
[![Docs](https://godoc.org/github.com/linkdata/bwlimit?status.svg)](https://godoc.org/github.com/linkdata/bwlimit)

# bwlimit

Go net.Conn bandwidth limiter.

Only depends on the standard library.

## Usage

`go get github.com/linkdata/bwlimit`

`Ticker` must be created with `bwlimit.NewTicker()`. The zero-value `Ticker` is not supported.
Limits are enforced in 100ms slices with fractional carry-over between slices, so very low limits are accurate over time but can still be bursty at slice boundaries.
After calling `Limiter.Stop()`, bandwidth metrics (`Count` and `Rate`) are no longer updated.

## Example

```go
import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/linkdata/bwlimit"
)

func main() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello world!"))
	}))
	defer srv.Close()

	// limit reads to 100 bytes/sec, unlimited writes
	lim := bwlimit.NewLimiter(100, 0)
	defer lim.Stop()

	// clone the default transport so this change is local to this client
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
}
```

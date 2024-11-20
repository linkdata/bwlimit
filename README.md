[![build](https://github.com/linkdata/bwlimit/actions/workflows/go.yml/badge.svg)](https://github.com/linkdata/bwlimit/actions/workflows/go.yml)
[![coverage](https://coveralls.io/repos/github/linkdata/bwlimit/badge.svg?branch=main)](https://coveralls.io/github/linkdata/bwlimit?branch=main)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/bwlimit)](https://goreportcard.com/report/github.com/linkdata/bwlimit)
[![Docs](https://godoc.org/github.com/linkdata/bwlimit?status.svg)](https://godoc.org/github.com/linkdata/bwlimit)

# bwlimit

Go net.Conn bandwidth limiter.

Only depends on the standard library.

## Usage

`go get github.com/linkdata/bwlimit`

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

	// wrap the default http transport DialContext
	tp := http.DefaultTransport.(*http.Transport)
	tp.DialContext = lim.Wrap(tp.DialContext)

	// make a request and time it
	now := time.Now()
	resp, err := http.Get(srv.URL)
	elapsed := time.Since(now)

	if err == nil {
		var body []byte
		if body, err = io.ReadAll(resp.Body); err == nil {
			fmt.Printf("%v %v %q\n", elapsed >= time.Second, lim.Reads.Count.Load() > 100, string(body))
		}
	}
}
```
package luadns

import (
	"errors"
	"net/http"
	"time"

	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	api "github.com/luadns/luadns-go"
)

// The API occasionally answers a valid request with a 5xx. A request that
// cannot create anything twice (anything but a POST) is repeated after a 500,
// 502, 503 or 504, with a pause that doubles up to maxRetryPause. A 429 means
// the request was not processed, so every request is repeated after one.
const (
	firstRetryPause = 2 * time.Second
	maxRetryPause   = 30 * time.Second
	maxRetries      = 6
)

// sleep is a variable so that tests can skip the pauses.
var sleep = time.Sleep

// call waits for the rate limiter and runs fn, repeating it as described above.
// idempotent reports whether fn may be repeated after a 5xx.
func (l *luadnsProvider) call(what string, idempotent bool, fn func() error) error {
	pause := firstRetryPause
	for attempt := 0; ; attempt++ {
		if err := l.rateLimiter.Wait(l.ctx); err != nil {
			return err
		}
		err := fn()
		if err == nil || attempt >= maxRetries || !retryable(err, idempotent) {
			return err
		}
		printer.Warnf("LUADNS: %s: %v, waiting %s\n", what, err, pause)
		sleep(pause)
		pause = min(2*pause, maxRetryPause)
	}
}

func retryable(err error, idempotent bool) bool {
	if _, ok := errors.AsType[*api.ErrTooManyRequests](err); ok {
		return true
	}
	if e, ok := errors.AsType[*api.ErrBadStatusCode](err); ok && idempotent {
		switch e.StatusCode {
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
	}
	return false
}

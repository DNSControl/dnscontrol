package transip

import (
	"log"
	"time"

	"github.com/transip/gotransip/v6/rest"
)

const (
	initialBackoff = time.Second * 20 // First delay duration
	maxBackoff     = time.Minute * 4  // Maximum backoff delay
)

// backoff is the amount of time to sleep if a 429 (or similar) is received.
// It is doubled after each use.
var (
	backoff = initialBackoff
)

func retryNeeded(err error) bool {
	if err == nil {
		return false // Not an error.
	}

	serr, ok := err.(*rest.Error)
	if !ok {
		return false // Not an error we know how to work with.
	}

	if serr.StatusCode == 200 {
		backoff = initialBackoff // Reset
		return false             // Success! No need to retry.
	}

	if serr.StatusCode != 429 {
		return false
	}

	// a simple exponential back-off
	log.Printf("Pausing due to ratelimit (%03d): %v seconds\n", serr.StatusCode, backoff)
	time.Sleep(backoff)
	backoff = min(backoff+(backoff/2), maxBackoff)

	return true // Request the API call be re-tried.
}

/*

TODO(tlim): Use X-Rate-Limit-Reset to optimize sleep time.

This is a rather lazy implementation.  A better implementation would examine
the X-Rate-Limit-Reset header and wait until that timestamp, if the timestamp
seems reasonable.  This implementation just does an exponential back-off.

This is what the documentation says:

> **Rate limit**
> The rate limit for this API uses a sliding window of 15 minutes. Within this window, a maximum of 1000 requests can be made per user.
>
> Every request returns the following headers, indicating the number of requests made within this window, the amount of requests remaining and the reset timestamp.
>
> ```text
> X-Rate-Limit-Limit: 1000
> X-Rate-Limit-Remaining: 650
> X-Rate-Limit-Reset: 1485875578
> ```

> When this rate limit is exceeded, the response contains an error with HTTP status code: `429: Too many requests`.

<https://api.transip.nl/rest/docs.html#header-rate-limit>

*/

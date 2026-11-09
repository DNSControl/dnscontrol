package cloudpress

import (
	"net/http"
	"testing"
	"time"
)

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		name       string
		base       time.Duration
		maxDelay   time.Duration
		retryAfter string
		want       []time.Duration
	}{
		{"default base", defaultRetryBaseDelay, defaultRetryMaxDelay, "", []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second}},
		{"10s base", 10 * time.Second, defaultRetryMaxDelay, "", []time.Duration{10 * time.Second, 20 * time.Second, 30 * time.Second, 30 * time.Second, 30 * time.Second, 30 * time.Second}},
		{"10s base 10m cap", 10 * time.Second, 10 * time.Minute, "", []time.Duration{10 * time.Second, 20 * time.Second, 40 * time.Second, 80 * time.Second, 160 * time.Second, 320 * time.Second}},
		{"base above cap", 45 * time.Second, defaultRetryMaxDelay, "", []time.Duration{45 * time.Second, 45 * time.Second}},
		{"retry-after seconds", defaultRetryBaseDelay, defaultRetryMaxDelay, "7", []time.Duration{7 * time.Second, 7 * time.Second}},
		{"retry-after capped", defaultRetryBaseDelay, defaultRetryMaxDelay, "120", []time.Duration{30 * time.Second}},
		{"retry-after http-date ignored", defaultRetryBaseDelay, defaultRetryMaxDelay, "Mon, 26 Oct 2026 12:00:00 GMT", []time.Duration{1 * time.Second, 2 * time.Second}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cloudpressProvider{retryBaseDelay: tt.base, retryMaxDelay: tt.maxDelay}
			resp := &http.Response{Header: http.Header{}}
			if tt.retryAfter != "" {
				resp.Header.Set("Retry-After", tt.retryAfter)
			}
			for attempt, want := range tt.want {
				if got := c.retryDelay(resp, attempt); got != want {
					t.Errorf("attempt %d: got %s, want %s", attempt, got, want)
				}
			}
		})
	}
}

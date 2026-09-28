package spaceship

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/namecheap/go-spaceship-sdk/client"
)

func rateLimitErr(retryAfter time.Duration) error {
	return &client.SpaceshipApiError{
		Status:     http.StatusTooManyRequests,
		Message:    "slow down",
		RetryAfter: retryAfter,
	}
}

func TestWithRetry_SucceedsFirstTry(t *testing.T) {
	var slept []time.Duration
	p := &spaceshipProvider{sleep: func(d time.Duration) { slept = append(slept, d) }}

	if err := p.withRetry(func() error { return nil }); err != nil {
		t.Fatalf("withRetry() error = %v", err)
	}
	if len(slept) != 0 {
		t.Fatalf("slept %v, want no waits", slept)
	}
}

func TestWithRetry_PassesThroughNon429(t *testing.T) {
	want := errors.New("nope")
	var slept []time.Duration
	p := &spaceshipProvider{sleep: func(d time.Duration) { slept = append(slept, d) }}

	err := p.withRetry(func() error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("withRetry() error = %v, want %v", err, want)
	}
	if len(slept) != 0 {
		t.Fatalf("slept %v, want no waits on non-429", slept)
	}
}

func TestWithRetry_RetriesRateLimitThenSucceeds(t *testing.T) {
	var slept []time.Duration
	p := &spaceshipProvider{sleep: func(d time.Duration) { slept = append(slept, d) }}

	calls := 0
	err := p.withRetry(func() error {
		calls++
		if calls == 1 {
			return rateLimitErr(5 * time.Second)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withRetry() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	wantWait := 5*time.Second + retryAfterMargin
	if len(slept) != 1 || slept[0] != wantWait {
		t.Fatalf("slept %v, want [%v]", slept, wantWait)
	}
}

func TestWithRetry_MissingRetryAfterUsesDefault(t *testing.T) {
	var slept []time.Duration
	p := &spaceshipProvider{sleep: func(d time.Duration) { slept = append(slept, d) }}

	calls := 0
	err := p.withRetry(func() error {
		calls++
		if calls == 1 {
			return rateLimitErr(0)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withRetry() error = %v", err)
	}
	if len(slept) != 1 || slept[0] != defaultRetryAfter {
		t.Fatalf("slept %v, want [%v]", slept, defaultRetryAfter)
	}
}

func TestWithRetry_DetectsWrapped429(t *testing.T) {
	var slept []time.Duration
	p := &spaceshipProvider{sleep: func(d time.Duration) { slept = append(slept, d) }}

	calls := 0
	err := p.withRetry(func() error {
		calls++
		if calls == 1 {
			return fmt.Errorf("get records: %w", rateLimitErr(2*time.Second))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withRetry() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestWithRetry_StopsWhenWaitWouldExceedBudget(t *testing.T) {
	var slept []time.Duration
	p := &spaceshipProvider{sleep: func(d time.Duration) { slept = append(slept, d) }}

	// 6m Retry-After plus the 1s margin fits once (6m1s < 10m) and not twice.
	limited := rateLimitErr(6 * time.Minute)
	err := p.withRetry(func() error { return limited })
	if !errors.Is(err, limited) {
		t.Fatalf("withRetry() error = %v, want the original 429", err)
	}
	if len(slept) != 1 {
		t.Fatalf("slept %d times, want 1 (second wait exceeds budget)", len(slept))
	}
	if slept[0] != 6*time.Minute+retryAfterMargin {
		t.Fatalf("first wait = %v, want %v", slept[0], 6*time.Minute+retryAfterMargin)
	}
}

func TestWithRetry_UnfittableWaitFailsFast(t *testing.T) {
	var slept []time.Duration
	p := &spaceshipProvider{sleep: func(d time.Duration) { slept = append(slept, d) }}

	limited := rateLimitErr(maxRateLimitWait)
	err := p.withRetry(func() error { return limited })
	if !errors.Is(err, limited) {
		t.Fatalf("withRetry() error = %v, want the original 429", err)
	}
	if len(slept) != 0 {
		t.Fatalf("slept %v, want no wait when Retry-After itself exceeds the budget", slept)
	}
}

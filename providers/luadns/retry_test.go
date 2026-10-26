package luadns

import (
	"context"
	"errors"
	"testing"
	"time"

	api "github.com/luadns/luadns-go"
	"golang.org/x/time/rate"
)

func TestCall(t *testing.T) {
	sleep = func(time.Duration) {}
	defer func() { sleep = time.Sleep }()

	err500 := &api.ErrBadStatusCode{StatusCode: 500}
	tests := []struct {
		name       string
		idempotent bool
		errs       []error // returned by successive attempts; nil after the list ends
		wantCalls  int
		wantErr    bool
	}{
		{"success", true, nil, 1, false},
		{"500 then success", true, []error{err500, err500}, 3, false},
		{"500 on POST is final", false, []error{err500}, 1, true},
		{"429 on POST is repeated", false, []error{&api.ErrTooManyRequests{}}, 2, false},
		{"404 is final", true, []error{&api.ErrBadStatusCode{StatusCode: 404}}, 1, true},
		{"other error is final", true, []error{errors.New("boom")}, 1, true},
		{"gives up", true, []error{err500, err500, err500, err500, err500, err500, err500, err500}, maxRetries + 1, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := &luadnsProvider{ctx: context.Background(), rateLimiter: rate.NewLimiter(rate.Inf, 1)}
			calls := 0
			err := l.call("test", tc.idempotent, func() error {
				calls++
				if calls <= len(tc.errs) {
					return tc.errs[calls-1]
				}
				return nil
			})
			if calls != tc.wantCalls {
				t.Errorf("calls = %d, want %d", calls, tc.wantCalls)
			}
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

package normalize

import (
	"fmt"
	"testing"
)

func TestCheckSoa(t *testing.T) {
	tests := []struct {
		isError bool
		expire  uint32
		minttl  uint32
		refresh uint32
		retry   uint32
		mbox    string
	}{
		// Expire
		{false, 123, 123, 123, 123, "foo.bar.com."},
		{true, 0, 123, 123, 123, "foo.bar.com."},
		// MinTTL
		{false, 123, 123, 123, 123, "foo.bar.com."},
		{true, 123, 0, 123, 123, "foo.bar.com."},
		// Refresh
		{false, 123, 123, 123, 123, "foo.bar.com."},
		{true, 123, 123, 0, 123, "foo.bar.com."},
		// Retry
		{false, 123, 123, 123, 123, "foo.bar.com."},
		{true, 123, 123, 123, 0, "foo.bar.com."},
		// Serial
		{false, 123, 123, 123, 123, "foo.bar.com."},
		{false, 123, 123, 123, 123, "foo.bar.com."},
		// MBox
		{true, 123, 123, 123, 123, ""},
		{false, 123, 123, 123, 123, "foo.bar.com."},
	}

	for _, test := range tests {
		experiment := fmt.Sprintf("%d %d %d %d %s", test.expire, test.minttl, test.refresh,
			test.retry, test.mbox)
		t.Run(experiment, func(t *testing.T) {
			err := checkSoa(test.expire, test.minttl, test.refresh, test.retry, test.mbox)
			checkError(t, err, test.isError, experiment)
		})
	}
}

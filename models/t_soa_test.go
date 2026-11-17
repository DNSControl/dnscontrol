package models_test

import (
	"fmt"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

func TestMakeSOA(t *testing.T) {
	tests := []struct {
		isError bool
		mbox    string
		refresh uint32
		retry   uint32
		expire  uint32
		minttl  uint32
	}{
		{false, "foo.bar.com.", 123, 123, 123, 123},
		// MBox
		{true, "", 123, 123, 123, 123},
		// Refresh
		{true, "foo.bar.com.", 0, 123, 123, 123},
		// Retry
		{true, "foo.bar.com.", 123, 0, 123, 123},
		// Expire
		{true, "foo.bar.com.", 123, 123, 0, 123},
		// MinTTL
		{true, "foo.bar.com.", 123, 123, 123, 0},
	}

	for _, tt := range tests {
		experiment := fmt.Sprintf("%q %d %d %d %d", tt.mbox, tt.refresh, tt.retry, tt.expire, tt.minttl)
		t.Run(experiment, func(t *testing.T) {
			// User input (dnsconfig.js) is checked.
			strict := nrc.Flags{StrictParamChecking: true}
			_, err := models.MakeSOA("example.com", nil, strict, "ns1.example.com.", tt.mbox, tt.refresh, tt.retry, tt.expire, tt.minttl)
			if (err != nil) != tt.isError {
				t.Errorf("6 args: got err=%v, wantErr=%v", err, tt.isError)
			}
			_, err = models.MakeSOA("example.com", nil, strict, "ns1.example.com.", tt.mbox, uint32(2026111700), tt.refresh, tt.retry, tt.expire, tt.minttl)
			if (err != nil) != tt.isError {
				t.Errorf("7 args: got err=%v, wantErr=%v", err, tt.isError)
			}
			// Provider data is accepted as-is.
			_, err = models.MakeSOA("example.com", nil, nrc.Flags{}, "ns1.example.com.", tt.mbox, uint32(2026111700), tt.refresh, tt.retry, tt.expire, tt.minttl)
			if err != nil {
				t.Errorf("unchecked: got err=%v, want nil", err)
			}
		})
	}
}

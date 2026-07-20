package vercel

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestCaaTargetContainsUnsupportedFields(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{
			name:    "simple domain",
			target:  "letsencrypt.org",
			wantErr: false,
		},
		{
			name:    "with cansignhttpexchanges",
			target:  "digicert.com; cansignhttpexchanges=yes",
			wantErr: false,
		},
		{
			name:    "with empty domain",
			target:  ";",
			wantErr: true,
		},
		{
			name:    "with validationmethods",
			target:  "letsencrypt.org; validationmethods=dns-01",
			wantErr: true,
		},
		{
			name:    "with accounturi",
			target:  "letsencrypt.org; accounturi=https://example.com",
			wantErr: true,
		},
		{
			name:    "with multiple params including allowed",
			target:  "letsencrypt.org; cansignhttpexchanges; validationmethods=dns-01",
			wantErr: true,
		},
		{
			name:    "with unknown param",
			target:  "letsencrypt.org; foo=bar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &models.RecordConfig{}
			rc.SetTarget(tt.target)
			if err := rejectifCaaTargetContainsUnsupportedFields(rc); (err != nil) != tt.wantErr {
				t.Errorf("caaTargetContainsUnsupportedFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

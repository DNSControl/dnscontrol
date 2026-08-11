package powerdns

import (
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/stretchr/testify/assert"
)

func TestAuditRecordsSvcbAutoHintOrder(t *testing.T) {
	tests := []struct {
		name    string
		params  string
		wantErr bool
	}{
		{
			name:    "sorted auto hints",
			params:  "alpn=h3,h2 ipv4hint=auto ipv6hint=auto",
			wantErr: false,
		},
		{
			name:    "ipv6hint before ipv4hint",
			params:  "alpn=h3,h2 ipv6hint=auto ipv4hint=auto",
			wantErr: true,
		},
		{
			name:    "non auto hints use regular validation path",
			params:  "alpn=h3,h2 ipv6hint=2001:db8::1 ipv4hint=192.0.2.1",
			wantErr: false,
		},
		{
			name:    "unknown params ignored",
			params:  "alpn=h3,h2 key65400=value ipv4hint=auto ipv6hint=auto",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := powerDNSSVCBRecord("HTTPS", tt.params)
			errs := AuditRecords(models.Records{record})

			if tt.wantErr {
				assert.Len(t, errs, 1)
				assert.True(t, strings.Contains(errs[0].Error(), "ipv4hint must appear before ipv6hint"))
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

func powerDNSSVCBRecord(rtype, params string) *models.RecordConfig {
	rc := &models.RecordConfig{
		Type:        rtype,
		SvcPriority: 1,
		SvcParams:   params,
	}
	rc.SetLabel("auto", "example.com")
	rc.MustSetTarget(".")
	return rc
}

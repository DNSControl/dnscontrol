package tencentdns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/stretchr/testify/assert"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

func TestNativeToRecord(t *testing.T) {
	domain := "example.com"

	tests := []struct {
		name     string
		input    *dnspod.RecordListItem
		expected *models.RecordConfig
	}{
		{
			name: "Basic A record",
			input: &dnspod.RecordListItem{
				Name:  new("@"),
				Type:  new("A"),
				Value: new("1.2.3.4"),
				TTL:   new(uint64(600)),
			},
			expected: &models.RecordConfig{
				Type: "A",
				TTL:  600,
			},
		},
		{
			name: "CNAME record",
			input: &dnspod.RecordListItem{
				Name:  new("www"),
				Type:  new("CNAME"),
				Value: new("target.example.com."),
				TTL:   new(uint64(300)),
			},
			expected: &models.RecordConfig{
				Type: "CNAME",
				TTL:  300,
			},
		},
		{
			name: "MX record",
			input: &dnspod.RecordListItem{
				Name:  new("@"),
				Type:  new("MX"),
				Value: new("mail.example.com."),
				TTL:   new(uint64(600)),
				MX:    new(uint64(10)),
			},
			expected: &models.RecordConfig{
				Type:         "MX",
				TTL:          600,
				MxPreference: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := nativeToRecord(tt.input, domain)
			if err != nil {
				t.Fatalf("nativeToRecord failed: %v", err)
			}
			assert.Equal(t, tt.expected.Type, rc.Type)
			assert.Equal(t, tt.expected.TTL, rc.TTL)
			if tt.expected.Type == "MX" {
				assert.Equal(t, tt.expected.MxPreference, rc.MxPreference)
			}
			expectedLabel := tt.expected.GetLabel()
			if expectedLabel == "" {
				expectedLabel = *tt.input.Name
			}
			assert.Equal(t, expectedLabel, rc.GetLabel())
		})
	}
}

func TestRecordToCreateRequest(t *testing.T) {
	domain := "example.com"
	rc := &models.RecordConfig{
		Type: "A",
		TTL:  600,
	}
	rc.SetLabel("test", domain)
	rc.SetTarget("1.1.1.1")

	req := recordToCreateRequest(rc)
	assert.Equal(t, "test", *req.SubDomain)
	assert.Equal(t, "A", *req.RecordType)
	assert.Equal(t, "1.1.1.1", *req.Value)
	assert.Equal(t, uint64(600), *req.TTL)
}

func TestRecordToCreateRequest_MX(t *testing.T) {
	domain := "example.com"
	rc := &models.RecordConfig{
		Type:         "MX",
		TTL:          600,
		MxPreference: 10,
	}
	rc.SetLabel("@", domain)
	rc.SetTarget("mail.example.com.")

	req := recordToCreateRequest(rc)
	assert.Equal(t, "@", *req.SubDomain)
	assert.Equal(t, "MX", *req.RecordType)
	assert.Equal(t, "mail.example.com.", *req.Value)
	assert.Equal(t, uint64(10), *req.MX)
}

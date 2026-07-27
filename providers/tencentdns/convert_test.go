package tencentdns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/stretchr/testify/assert"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

func TestNativeToRecord(t *testing.T) {
	tests := []struct {
		name     string
		input    *dnspod.RecordListItem
		expected string
	}{
		{
			name: "Basic A record",
			input: &dnspod.RecordListItem{
				Name:  new("@"),
				Type:  new("A"),
				Value: new("1.2.3.4"),
				TTL:   new(uint64(600)),
			},
			expected: "@ 600 IN A 1.2.3.4",
		},
		{
			name: "CNAME record",
			input: &dnspod.RecordListItem{
				Name:  new("www"),
				Type:  new("CNAME"),
				Value: new("target.example.com."),
				TTL:   new(uint64(300)),
			},
			expected: "www 300 IN CNAME target.example.com.",
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
			expected: "@ 600 IN MX 10 mail.example.com.",
		},
	}

	testDomain := models.MustNewDomainConfig("example.com")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := nativeToRecord(tt.input, testDomain)
			if err != nil {
				t.Fatalf("nativeToRecord failed: %v", err)
			}
			got := rc.LineString()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNativeToRecordPreservesProviderMetadata(t *testing.T) {
	testDomain := models.MustNewDomainConfig("example.com")
	input := &dnspod.RecordListItem{
		Name:   new("www"),
		Type:   new("A"),
		Value:  new("1.2.3.4"),
		TTL:    new(uint64(600)),
		Line:   new("电信"),
		LineId: new("10=1"),
		Weight: new(uint64(80)),
	}

	rc, err := nativeToRecord(input, testDomain)
	if err != nil {
		t.Fatalf("nativeToRecord failed: %v", err)
	}

	assert.Equal(t, "电信", rc.Metadata[metaRecordLine])
	assert.Equal(t, "10=1", rc.Metadata[metaRecordLineID])
	assert.Equal(t, "80", rc.Metadata[metaRecordWeight])
}

func TestNativeToRecordPreservesDisabledWeight(t *testing.T) {
	testDomain := models.MustNewDomainConfig("example.com")
	input := &dnspod.RecordListItem{
		Name:   new("www"),
		Type:   new("A"),
		Value:  new("1.2.3.4"),
		TTL:    new(uint64(600)),
		Weight: new(uint64(0)),
	}

	rc, err := nativeToRecord(input, testDomain)
	if assert.NoError(t, err) {
		assert.Equal(t, "0", rc.Metadata[metaRecordWeight])
	}
}

func TestRecordToCreateRequest(t *testing.T) {
	testDomain := models.MustNewDomainConfig("example.com")
	rc := testDomain.MustNewRecordConfig("test", 600, "A", "1.1.1.1")

	req := recordToCreateRequest(rc)
	assert.Equal(t, "test", *req.SubDomain)
	assert.Equal(t, "A", *req.RecordType)
	assert.Equal(t, defaultRecordLine, *req.RecordLine)
	assert.Nil(t, req.RecordLineId)
	assert.Nil(t, req.Weight)
	assert.Equal(t, "1.1.1.1", *req.Value)
	assert.Equal(t, uint64(600), *req.TTL)
}

func TestRecordToCreateRequestWithWeight(t *testing.T) {
	testDomain := models.MustNewDomainConfig("example.com")
	rc := testDomain.MustNewRecordConfig("test", 600, "A", "1.1.1.1")
	rc.Metadata = map[string]string{
		metaRecordWeight: "80",
	}

	req := recordToCreateRequest(rc)
	assert.Equal(t, uint64(80), *req.Weight)
}

func TestRecordToCreateRequestWithLine(t *testing.T) {
	testDomain := models.MustNewDomainConfig("example.com")
	rc := testDomain.MustNewRecordConfig("test", 600, "A", "1.1.1.1")
	rc.Metadata = map[string]string{
		metaRecordLine: "电信",
	}

	req := recordToCreateRequest(rc)
	assert.Equal(t, "电信", *req.RecordLine)
	assert.Nil(t, req.RecordLineId)
}

func TestRecordToCreateRequestWithLineID(t *testing.T) {
	testDomain := models.MustNewDomainConfig("example.com")
	rc := testDomain.MustNewRecordConfig("test", 600, "A", "1.1.1.1")
	rc.Metadata = map[string]string{
		metaRecordLineID: "10=1",
	}

	req := recordToCreateRequest(rc)
	assert.Equal(t, defaultRecordLine, *req.RecordLine)
	assert.Equal(t, "10=1", *req.RecordLineId)
}

func TestRecordToModifyRequestWithLineLineIDAndWeight(t *testing.T) {
	testDomain := models.MustNewDomainConfig("example.com")
	rc := testDomain.MustNewRecordConfig("test", 600, "A", "1.1.1.1")
	rc.Metadata = map[string]string{
		metaRecordLine:   "电信",
		metaRecordLineID: "10=1",
		metaRecordWeight: "25",
	}

	req := recordToModifyRequest(rc, 42, nil)
	assert.Equal(t, uint64(42), *req.RecordId)
	assert.Equal(t, "电信", *req.RecordLine)
	assert.Equal(t, "10=1", *req.RecordLineId)
	assert.Equal(t, uint64(25), *req.Weight)
}

func TestRecordToModifyRequestClearsRemovedWeight(t *testing.T) {
	testDomain := models.MustNewDomainConfig("example.com")
	previous := testDomain.MustNewRecordConfig("test", 600, "A", "1.1.1.1")
	previous.Metadata = map[string]string{
		metaRecordWeight: "80",
	}

	desired := testDomain.MustNewRecordConfig("test", 600, "A", "1.1.1.1")

	req := recordToModifyRequest(desired, 42, previous)
	assert.Equal(t, uint64(0), *req.Weight)
}

func TestRecordToCreateRequest_MX(t *testing.T) {
	testDomain := models.MustNewDomainConfig("example.com")
	rc := testDomain.MustNewRecordConfig("@", 600, "MX", 10, "mail.example.com.")

	req := recordToCreateRequest(rc)
	assert.Equal(t, "@", *req.SubDomain)
	assert.Equal(t, "MX", *req.RecordType)
	assert.Equal(t, "mail.example.com.", *req.Value)
	assert.Equal(t, uint64(10), *req.MX)
}

// func expectedRecord(rtype string, ttl uint32, mxPreference uint16) *models.RecordConfig {
// 	dc := models.MustNewDomainConfig("example.com")
// 	rc := dc.MustNewRecordConfig("@", ttl, rtype,
// )
// 	rc := new(models.RecordConfig)
// 	rc.Type = rtype
// 	rc.TTL = ttl
// 	f := rc.AsMX()
// 	f.Preference = mxPreference
// 	rc.SetRDATA(f)
// 	return rc
// }

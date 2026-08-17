package tencentdns

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/stretchr/testify/assert"
)

func TestAuditRecords(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	mxNull, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeMX, 10, ".")
	assert.NoError(t, err)

	txtEmpty, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeTXT, "")
	assert.NoError(t, err)

	txtSingleQuote, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeTXT, "quo'te")
	assert.NoError(t, err)

	txtDoubleQuote, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeTXT, `in"side`)
	assert.NoError(t, err)

	txtBackslash, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeTXT, `foo\bar`)
	assert.NoError(t, err)

	txtTrailingSpace, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeTXT, "trailingws ")
	assert.NoError(t, err)

	srvNull, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeSRV, 0, 0, 1, ".")
	assert.NoError(t, err)

	srvEmpty, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeSRV, 0, 0, 1, "")
	assert.NoError(t, err)

	validA, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeA, "1.2.3.4")
	assert.NoError(t, err)

	errs := AuditRecords(models.Records{mxNull, txtEmpty, txtSingleQuote, txtDoubleQuote, txtBackslash, txtTrailingSpace, srvNull, srvEmpty, validA})

	assert.Len(t, errs, 8)
	assert.Contains(t, errs[0].Error(), "mx has null target")
	assert.Contains(t, errs[1].Error(), "txtstring is empty")
	assert.Contains(t, errs[2].Error(), "txtstring contains single-quotes")
	assert.Contains(t, errs[3].Error(), "txtstring contains doublequotes")
	assert.Contains(t, errs[4].Error(), "txtstring contains backslashes")
	assert.Contains(t, errs[5].Error(), "txtstring ends with space")
	assert.Contains(t, errs[6].Error(), "srv has empty target")
	assert.Contains(t, errs[7].Error(), "srv has empty target")
}

func TestAuditRecordsValidatesWeight(t *testing.T) {
	tests := []struct {
		name      string
		weight    string
		wantError bool
	}{
		{name: "unset"},
		{name: "minimum", weight: "0"},
		{name: "maximum", weight: "100"},
		{name: "negative", weight: "-1", wantError: true},
		{name: "too large", weight: "101", wantError: true},
		{name: "not an integer", weight: "heavy", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dc := models.MustNewDomainConfig("example.com")
			rc := dc.MustNewRecordConfig("@", 0, "A", "1.2.3.4")
			rc.Metadata = map[string]string{
				metaRecordWeight: tc.weight,
			}

			errs := AuditRecords(models.Records{rc})
			if tc.wantError {
				if assert.Len(t, errs, 1) {
					assert.Contains(t, errs[0].Error(), metaRecordWeight)
				}
				return
			}
			assert.Empty(t, errs)
		})
	}
}

func TestTargetConstraint(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{
			name:   "ascii target",
			target: "www.example.com.",
		},
		{
			name:   "chinese target",
			target: "xn--55qx5d.",
		},
		{
			name:    "non-chinese idn target",
			target:  "xn--ndaaa.com.",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := models.MustNewDomainConfig("example.com")
			rc := dc.MustNewRecordConfig("a", 0, dnsv2.TypeCNAME, tt.target)

			err := targetConstraint(rc)
			if (err != nil) != tt.wantErr {
				t.Fatalf("targetConstraint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuditRecordsRejectsNonChineseIDNCNAMETarget(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("a", 0, dnsv2.TypeCNAME, "xn--ndaaa.com.")

	errs := AuditRecords(models.Records{rc})
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "target contains non-ASCII characters")
}

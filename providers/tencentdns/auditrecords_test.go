package tencentdns

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/stretchr/testify/assert"
)

func TestAuditRecords(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	mxNull, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeMX, 10, ".")
	assert.NoError(t, err)

	txtEmpty, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeTXT, "")
	assert.NoError(t, err)

	srvNull, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeSRV, 0, 0, 1, ".")
	assert.NoError(t, err)

	srvEmpty, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeSRV, 0, 0, 1, "")
	assert.NoError(t, err)

	validA, err := dc.NewRecordConfig("foo", 0, dnsv2.TypeA, "1.2.3.4")
	assert.NoError(t, err)

	errs := AuditRecords(models.Records{mxNull, txtEmpty, srvNull, srvEmpty, validA})

	assert.Len(t, errs, 4)
	assert.Contains(t, errs[0].Error(), "mx has null target")
	assert.Contains(t, errs[1].Error(), "txtstring is empty")
	assert.Contains(t, errs[2].Error(), "srv has empty target")
	assert.Contains(t, errs[3].Error(), "srv has empty target")
}

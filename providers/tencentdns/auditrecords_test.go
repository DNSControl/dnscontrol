package tencentdns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/stretchr/testify/assert"
)

func TestAuditRecords(t *testing.T) {
	mxNull := &models.RecordConfig{Type: "MX"}
	assert.NoError(t, mxNull.SetTargetMX(0, "."))

	txtEmpty := &models.RecordConfig{Type: "TXT"}
	assert.NoError(t, txtEmpty.SetTargetTXT(""))

	srvNull := &models.RecordConfig{Type: "SRV"}
	assert.NoError(t, srvNull.SetTargetSRV(0, 0, 1, "."))

	srvEmpty := &models.RecordConfig{Type: "SRV"}
	assert.NoError(t, srvEmpty.SetTargetSRV(0, 0, 1, ""))

	validA := &models.RecordConfig{Type: "A"}
	validA.SetTarget("1.2.3.4")

	errs := AuditRecords(models.Records{mxNull, txtEmpty, srvNull, srvEmpty, validA})

	assert.Len(t, errs, 4)
	assert.Contains(t, errs[0].Error(), "mx has null target")
	assert.Contains(t, errs[1].Error(), "txtstring is empty")
	assert.Contains(t, errs[2].Error(), "srv has empty target")
	assert.Contains(t, errs[3].Error(), "srv has empty target")
}

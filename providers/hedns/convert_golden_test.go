package hedns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

var testDomain = providergolden.Domain("HEDNS")

func TestRecordToRCGolden(t *testing.T) {
	providergolden.CheckToRC(t, "hedns_recordtorc", testDomain,
		func(dc *models.DomainConfig, native Record) ([]*models.RecordConfig, error) {
			rc, err := recordToRC(dc, native)
			return []*models.RecordConfig{rc}, err
		})
}

package hedns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestRecordToRCGolden(t *testing.T) {
	providergolden.CheckToRC(t, "recordToRC",
		func(dc *models.DomainConfig, native Record) ([]*models.RecordConfig, error) {
			rc, err := recordToRC(dc, native)
			return []*models.RecordConfig{rc}, err
		})
}

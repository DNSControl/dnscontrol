package namedotcom

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"github.com/namedotcom/go/namecom"
)

func TestToRecordGolden(t *testing.T) {
	providergolden.CheckToRC(t, "namedotcom_torecord",
		func(dc *models.DomainConfig, native namecom.Record) ([]*models.RecordConfig, error) {
			rc, err := toRecord(&native, dc)
			return []*models.RecordConfig{rc}, err
		})
}

package netlify

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestToRecordConfigGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRecordConfig",
		func(dc *models.DomainConfig, native dnsRecord) ([]*models.RecordConfig, error) {
			rc, err := toRecordConfig(dc, &native)
			return []*models.RecordConfig{rc}, err
		})
}

func TestToReqGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toReq",
		func(_ *models.DomainConfig, records models.Records) (*dnsRecordCreate, error) {
			return toReq(records[0]), nil
		})
}

package cloudpress

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestToRecordConfigGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRecordConfig",
		func(dc *models.DomainConfig, native record) (models.Records, error) {
			rc, err := toRecordConfig(dc, &native)
			return models.Records{rc}, err
		})
}

func TestFromRecordConfigGolden(t *testing.T) {
	providergolden.CheckToNative(t, "fromRecordConfig",
		func(_ *models.DomainConfig, records models.Records) (*record, error) {
			return fromRecordConfig(records[0]), nil
		})
}

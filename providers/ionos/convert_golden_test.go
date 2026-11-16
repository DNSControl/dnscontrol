package ionos

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestToRCGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRC",
		func(dc *models.DomainConfig, native record) (models.Records, error) {
			rc, err := toRC(dc, native)
			return models.Records{rc}, err
		})
}

func TestToNativeGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toNative",
		func(_ *models.DomainConfig, records models.Records) (record, error) {
			return toNative(records[0]), nil
		})
}

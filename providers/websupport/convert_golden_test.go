package websupport

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestToRecordConfigGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRecordConfig",
		func(dc *models.DomainConfig, native nativeRecord) (models.Records, error) {
			rc, err := toRecordConfig(dc, native)
			return models.Records{rc}, err
		})
}

func TestToNativeGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toNative", func(_ *models.DomainConfig, records models.Records) (nativeRecord, error) {
		return toNative(records[0])
	})
}

func TestConversionRoundTrip(t *testing.T) {
	providergolden.CheckRoundTrip(t, "toNative", func(_ *models.DomainConfig, records models.Records) (nativeRecord, error) {
		return toNative(records[0])
	},
		func(dc *models.DomainConfig, native nativeRecord) (models.Records, error) {
			rc, err := toRecordConfig(dc, native)
			return models.Records{rc}, err
		})
}

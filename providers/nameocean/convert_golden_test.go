package nameocean

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestToRCGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRC",
		func(dc *models.DomainConfig, native nativeRecord) (models.Records, error) {
			rc, err := toRC(dc, native)
			return models.Records{rc}, err
		})
}

func TestToNativeGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toNative",
		func(dc *models.DomainConfig, records models.Records) (nativeRecord, error) {
			return toNative(dc, records[0])
		})
}

func TestRoundTrip(t *testing.T) {
	providergolden.CheckRoundTrip(t, "toNative",
		func(dc *models.DomainConfig, records models.Records) (nativeRecord, error) {
			return toNative(dc, records[0])
		},
		func(dc *models.DomainConfig, native nativeRecord) (models.Records, error) {
			rc, err := toRC(dc, native)
			return models.Records{rc}, err
		})
}

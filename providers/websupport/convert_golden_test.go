package websupport

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestToRecordConfigGolden(t *testing.T) {
	providergolden.CheckToRC(t, "websupport_torecordconfig", testDomain,
		func(dc *models.DomainConfig, native nativeRecord) ([]*models.RecordConfig, error) {
			rc, err := toRecordConfig(dc, native)
			return []*models.RecordConfig{rc}, err
		})
}

func TestToNativeGolden(t *testing.T) {
	providergolden.CheckToNative(t, "websupport_tonative", testDomain, toNative)
}

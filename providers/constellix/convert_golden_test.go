package constellix

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestToRCGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRC", toRC)
}

func TestToNativeGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toNative", func(_ *models.DomainConfig, records models.Records) (recordRequest, error) {
		return toNative(records)
	})
}

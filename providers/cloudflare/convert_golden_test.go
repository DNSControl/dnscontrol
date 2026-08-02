package cloudflare

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"github.com/cloudflare/cloudflare-go"
)

const testDomain = "example.com"

func TestNativeToRecordGolden(t *testing.T) {
	c := &cloudflareProvider{}
	providergolden.CheckToRC(t, "cloudflare_nativetorecord", testDomain,
		func(dc *models.DomainConfig, native cloudflare.DNSRecord) ([]*models.RecordConfig, error) {
			rc, err := c.nativeToRecord(dc, native)
			return []*models.RecordConfig{rc}, err
		})
}

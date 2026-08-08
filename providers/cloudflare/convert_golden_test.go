package cloudflare

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"github.com/cloudflare/cloudflare-go"
)

func TestNativeToRecordGolden(t *testing.T) {
	c := &cloudflareProvider{}
	providergolden.CheckToRC(t, "nativeToRecord",
		func(dc *models.DomainConfig, native cloudflare.DNSRecord) ([]*models.RecordConfig, error) {
			if native.Type == "" {
				// TODO(tlim): Figure out why we're getting natives with no type.  Something to do with CLOUDFLAREAPI_SINGLE_REDIRECT.
				return nil, nil
			}
			rc, err := c.nativeToRecord(dc, native)
			return []*models.RecordConfig{rc}, err
		})
}

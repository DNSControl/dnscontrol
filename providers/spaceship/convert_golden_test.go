package spaceship

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"github.com/namecheap/go-spaceship-sdk/client"
)

func TestToRCGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRC",
		func(dc *models.DomainConfig, native client.DNSRecord) (models.Records, error) {
			rc, err := toRC(dc, native)
			return models.Records{rc}, err
		})
}

func TestToNativeGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toNative", func(_ *models.DomainConfig, records models.Records) (client.DNSRecord, error) {
		return toNative(records[0])
	})
}

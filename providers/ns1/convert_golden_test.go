package ns1

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

func TestConvertGolden(t *testing.T) {
	providergolden.CheckToRC(t, "convert",
		func(dc *models.DomainConfig, native dns.ZoneRecord) (models.Records, error) {
			return convert(&native, dc)
		})
}

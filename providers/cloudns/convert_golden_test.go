package cloudns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

var testDomain = providergolden.Domain("CLOUDNS")

func TestToRcGolden(t *testing.T) {
	providergolden.CheckToRC(t, "cloudns_torc", testDomain,
		func(dc *models.DomainConfig, native domainRecord) ([]*models.RecordConfig, error) {
			rc, err := toRc(dc, &native)
			return []*models.RecordConfig{rc}, err
		})
}

func TestToReqGolden(t *testing.T) {
	providergolden.CheckToNative(t, "cloudns_toreq", testDomain, toReq)
}

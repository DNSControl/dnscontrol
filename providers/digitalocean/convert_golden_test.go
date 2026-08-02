package digitalocean

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"github.com/digitalocean/godo"
)

const testDomain = "example.com"

func TestToRcGolden(t *testing.T) {
	providergolden.CheckToRC(t, "digitalocean_torc", testDomain,
		func(dc *models.DomainConfig, native godo.DomainRecord) ([]*models.RecordConfig, error) {
			rc, err := toRc(dc, &native)
			return []*models.RecordConfig{rc}, err
		})
}

func TestToReqGolden(t *testing.T) {
	providergolden.CheckToNative(t, "digitalocean_toreq", testDomain,
		func(rc *models.RecordConfig) (*godo.DomainRecordEditRequest, error) {
			return toReq(rc), nil
		})
}

package digitalocean

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"github.com/digitalocean/godo"
)

func TestToRcGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRc",
		func(dc *models.DomainConfig, native godo.DomainRecord) (models.Records, error) {
			rc, err := toRc(dc, &native)
			return models.Records{rc}, err
		})
}

func TestToReqGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toReq",
		func(_ *models.DomainConfig, records models.Records) (*godo.DomainRecordEditRequest, error) {
			return toReq(records[0]), nil
		})
}

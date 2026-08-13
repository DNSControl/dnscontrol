package porkbun

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestToRcGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRc",
		func(dc *models.DomainConfig, native domainRecord) (models.Records, error) {
			rc, err := toRc(dc, &native)
			return models.Records{rc}, err
		})
}

func TestToReqGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toReq", func(_ *models.DomainConfig, records models.Records) (requestParams, error) {
		return toReq(records[0])
	})
}

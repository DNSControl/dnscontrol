package packetframe

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestToRcGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRc",
		func(dc *models.DomainConfig, native domainRecord) ([]*models.RecordConfig, error) {
			rc, err := toRc(dc, &native)
			return []*models.RecordConfig{rc}, err
		})
}

func TestToReqGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toReq",
		func(_ *models.DomainConfig, records models.Records) (*domainRecord, error) {
			return toReq("zone-1", records[0])
		})
}

func TestConversionRoundTrip(t *testing.T) {
	providergolden.CheckRoundTrip(t, "toReq",
		func(_ *models.DomainConfig, records models.Records) (*domainRecord, error) {
			return toReq("zone-1", records[0])
		},
		func(dc *models.DomainConfig, native *domainRecord) ([]*models.RecordConfig, error) {
			rc, err := toRc(dc, native)
			return []*models.RecordConfig{rc}, err
		})
}

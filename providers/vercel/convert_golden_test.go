package vercel

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

var testDomain = providergolden.Domain("VERCEL")

func TestVercelRecordToRCGolden(t *testing.T) {
	providergolden.CheckToRC(t, "vercel_vercelrecordtorc", testDomain,
		func(dc *models.DomainConfig, native DNSRecord) ([]*models.RecordConfig, error) {
			rc, err := vercelRecordToRC(dc, native)
			return []*models.RecordConfig{rc}, err
		})
}

func TestToVercelCreateRequestGolden(t *testing.T) {
	providergolden.CheckToNative(t, "vercel_tovercelcreaterequest", testDomain,
		func(rc *models.RecordConfig) (createDNSRecordRequest, error) {
			return toVercelCreateRequest(testDomain, rc)
		})
}

func TestToVercelUpdateRequestGolden(t *testing.T) {
	providergolden.CheckToNative(t, "vercel_tovercelupdaterequest", testDomain, toVercelUpdateRequest)
}

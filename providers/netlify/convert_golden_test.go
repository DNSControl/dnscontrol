package netlify

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

const testDomain = "example.com"

func TestToRecordConfigGolden(t *testing.T) {
	providergolden.CheckToRC(t, "netlify_torecordconfig", testDomain,
		func(dc *models.DomainConfig, native dnsRecord) ([]*models.RecordConfig, error) {
			rc, err := toRecordConfig(dc, &native)
			return []*models.RecordConfig{rc}, err
		})
}

func TestToReqGolden(t *testing.T) {
	providergolden.CheckToNative(t, "netlify_toreq", testDomain,
		func(rc *models.RecordConfig) (*dnsRecordCreate, error) {
			return toReq(rc), nil
		})
}

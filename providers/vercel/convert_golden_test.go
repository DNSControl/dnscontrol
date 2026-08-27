package vercel

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestVercelRecordToRCGolden(t *testing.T) {
	providergolden.CheckToRC(t, "vercelRecordToRC",
		func(dc *models.DomainConfig, native DNSRecord) (models.Records, error) {
			rc, err := vercelRecordToRC(dc, native)
			return models.Records{rc}, err
		})
}

func TestToVercelCreateRequestGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toVercelCreateRequest",
		func(dc *models.DomainConfig, records models.Records) (createDNSRecordRequest, error) {
			return toVercelCreateRequest(dc.Name, records[0])
		})
}

func TestToVercelUpdateRequestGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toVercelUpdateRequest", func(_ *models.DomainConfig, records models.Records) (updateDNSRecordRequest, error) {
		return toVercelUpdateRequest(records[0])
	})
}

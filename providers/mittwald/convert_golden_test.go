package mittwald

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	mwdns "github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/dnsv2"
)

func TestToRCGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRC", func(dc *models.DomainConfig, native mwdns.Zone) (models.Records, error) {
		return toRC(dc, native)
	})
}

func TestToNativeGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toNative", func(_ *models.DomainConfig, records models.Records) (map[domainclientv2.UpdateRecordSetRequestPathRecordSet]domainclientv2.UpdateRecordSetRequestBody, error) {
		return toNative(records)
	})
}

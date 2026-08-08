package cnr

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestCreateRecordStringGolden(t *testing.T) {
	domain := providergolden.RecordedDomain(t)
	providergolden.CheckToNative(t, "cnr_createrecordstring",
		func(rc *models.RecordConfig) (string, error) {
			return (&Client{}).createRecordString(rc, domain)
		})
}

package cnr

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

var testDomain = providergolden.Domain("CNR")

func TestCreateRecordStringGolden(t *testing.T) {
	providergolden.CheckToNative(t, "cnr_createrecordstring", testDomain,
		func(rc *models.RecordConfig) (string, error) {
			return (&Client{}).createRecordString(rc, testDomain)
		})
}

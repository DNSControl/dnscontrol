package gandiv5

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

var testDomain = providergolden.Domain("GANDI_V5")

func TestNativeToRecordsGolden(t *testing.T) {
	providergolden.CheckToRC(t, "gandiv5_nativetorecords", testDomain, nativeToRecords)
}

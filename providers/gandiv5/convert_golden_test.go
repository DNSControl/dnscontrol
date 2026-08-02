package gandiv5

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

const testDomain = "example.com"

func TestNativeToRecordsGolden(t *testing.T) {
	providergolden.CheckToRC(t, "gandiv5_nativetorecords", testDomain, nativeToRecords)
}

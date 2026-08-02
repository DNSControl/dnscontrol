package transip

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

const testDomain = "example.com"

func TestRecordToNativeGolden(t *testing.T) {
	providergolden.CheckToNative(t, "transip_recordtonative", testDomain, recordToNative)
}

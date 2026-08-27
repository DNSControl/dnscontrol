package gandiv5

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestNativeToRecordsGolden(t *testing.T) {
	providergolden.CheckToRC(t, "nativeToRecords", nativeToRecords)
}

package akamaiedgedns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/v12/pkg/dns"
)

func TestRcToRsGolden(t *testing.T) {
	provider := &edgeDNSProvider{}
	providergolden.CheckToNative(t, "rcToRs",
		func(_ *models.DomainConfig, records models.Records) (*dns.RecordBody, error) {
			return provider.rcToRs(records)
		})
}

func TestNativeToRecordsGolden(t *testing.T) {
	providergolden.CheckToRC(t, "nativeToRecords", nativeToRecords)
}

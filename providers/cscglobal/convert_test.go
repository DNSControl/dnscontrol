package cscglobal

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestNativeRecordConversion(t *testing.T) {
	t.Parallel()

	dc := models.MustNewDomainConfig("example.com")
	a := nativeToRecordA(nativeRecordA{Key: "www", Value: "192.0.2.1"}, dc, 600)
	if a.TTL != 600 || a.GetRDATA().String() != "192.0.2.1" {
		t.Errorf("unexpected A conversion: TTL=%d RDATA=%q", a.TTL, a.GetRDATA().String())
	}

	srv := nativeToRecordSRV(nativeRecordSRV{
		Key: "_sip._tcp", Value: "service.example.net.", TTL: 300,
		Priority: 1, Weight: 2, Port: 443,
	}, dc, 600)
	if got := srv.GetRDATA().String(); got != "1 2 443 service.example.net." {
		t.Errorf("unexpected SRV conversion: %q", got)
	}
}

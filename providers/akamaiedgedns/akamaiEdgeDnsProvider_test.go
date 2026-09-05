package akamaiedgedns

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
)

func TestPreprocessConfigConvertsAliasRDATA(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	apex := dc.MustNewRecordConfig("@", 300, privatetypes.TypeALIAS, "target.example.")
	nonApex := dc.MustNewRecordConfig("www", 300, privatetypes.TypeALIAS, "target.example.")
	dc.Records = models.Records{apex, nonApex}

	p := &edgeDNSProvider{}
	if err := p.preprocessConfig(dc); err != nil {
		t.Fatalf("preprocessConfig: %v", err)
	}

	if apex.Type != "AKAMAITLC" || apex.TypeNum != privatetypes.TypeAKAMAITLC {
		t.Fatalf("apex type = %s/%d, want AKAMAITLC/%d", apex.Type, apex.TypeNum, privatetypes.TypeAKAMAITLC)
	}
	apexRDATA := apex.AsAKAMAITLC()
	if apexRDATA.AnswerType != "DUAL" || apexRDATA.Target != "target.example." {
		t.Fatalf("apex RDATA = %#v", apexRDATA)
	}

	if nonApex.Type != "CNAME" || nonApex.TypeNum != dnsv2.TypeCNAME {
		t.Fatalf("non-apex type = %s/%d, want CNAME/%d", nonApex.Type, nonApex.TypeNum, dnsv2.TypeCNAME)
	}
	if got := nonApex.AsCNAME().Target; got != "target.example." {
		t.Fatalf("non-apex target = %q, want %q", got, "target.example.")
	}

	rs, err := p.rcToRs(models.Records{apex})
	if err != nil {
		t.Fatalf("rcToRs: %v", err)
	}
	if len(rs.Target) != 1 || rs.Target[0] != "DUAL target.example." {
		t.Fatalf("Akamai targets = %#v, want %#v", rs.Target, []string{"DUAL target.example."})
	}
}

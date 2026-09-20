package spaceship

import (
	"os"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"github.com/namecheap/go-spaceship-sdk/client"
)

func sampleRecords() []client.DNSRecord {
	flag := 0
	pref := 10
	prio := 10
	weight := 20
	svcPrio := 1
	usage := 3
	selector := 1
	matching := 1

	return []client.DNSRecord{
		{Type: "A", Name: "@", TTL: 600, Address: "192.0.2.1"},
		{Type: "A", Name: "www", TTL: 600, Address: "192.0.2.2"},
		{Type: "AAAA", Name: "www", TTL: 600, Address: "2001:db8::1"},
		{Type: "CNAME", Name: "alias", TTL: 600, CName: "www.example.com"},
		{Type: "ALIAS", Name: "docs", TTL: 600, AliasName: "origin.example.com"},
		{Type: "MX", Name: "@", TTL: 3600, Preference: &pref, Exchange: "mail.example.com"},
		{Type: "TXT", Name: "@", TTL: 600, Value: "v=spf1 include:_spf.example.net -all"},
		{Type: "NS", Name: "sub", TTL: 3600, Nameserver: "ns1.example.net"},
		{Type: "PTR", Name: "ptr", TTL: 600, Pointer: "host.example.com"},
		{Type: "CAA", Name: "@", TTL: 600, Flag: &flag, Tag: "issue", Value: "letsencrypt.org"},
		{
			Type: "SRV", Name: "@", TTL: 600,
			Service: "_sip", Protocol: "_tcp",
			Priority: &prio, Weight: &weight,
			Port: client.NewIntPortValue(5060), Target: "sip.example.com",
		},
		{
			Type: "HTTPS", Name: "@", TTL: 600,
			SvcPriority: &svcPrio, TargetName: ".", SvcParams: "alpn=h3,h2",
		},
		{
			Type: "SVCB", Name: "svc", TTL: 600,
			Port: client.NewStringPortValue("_8443"), Scheme: "_https",
			SvcPriority: &svcPrio, TargetName: "svc.example.net", SvcParams: "alpn=h2",
		},
		{
			Type: "TLSA", Name: "mail", TTL: 600,
			Port: client.NewStringPortValue("_25"), Protocol: "_tcp",
			Usage: &usage, Selector: &selector, Matching: &matching,
			AssociationData: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
	}
}

func TestRoundTripRecords(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	for _, native := range sampleRecords() {
		t.Run(native.Type+"/"+native.Name, func(t *testing.T) {
			rc, err := toRC(dc, native)
			if err != nil {
				t.Fatalf("toRC: %v", err)
			}
			got, err := toNative(rc)
			if err != nil {
				t.Fatalf("toNative: %v", err)
			}
			if client.RecordKey(got) != client.RecordKey(strippedGroup(native)) {
				t.Fatalf("round trip key\n got %s\nwant %s", client.RecordKey(got), client.RecordKey(native))
			}
			if got.TTL != int(clampTTL(uint32(native.TTL))) && got.TTL != native.TTL {
				t.Fatalf("TTL = %d, native %d", got.TTL, native.TTL)
			}
		})
	}
}

func TestWriteGolden(t *testing.T) {
	if os.Getenv("WRITE_GOLDEN") == "" {
		t.Skip("set WRITE_GOLDEN=1 to regenerate test_data")
	}
	dc := models.MustNewDomainConfig("example.com")
	recorder := providergolden.NewRecorder()
	observer := recorder.ForDomain(dc.Name)
	for _, native := range sampleRecords() {
		before := observer.BeginToRC("toRC", native)
		rc, err := toRC(dc, native)
		observer.EndToRC("toRC", before, native, models.Records{rc}, err)
		if err != nil {
			t.Fatal(err)
		}
		// Drop Original so ToNative recordings match parse-from-text fixtures.
		rc.Original = nil
		beforeNative := observer.BeginToNative("toNative", models.Records{rc})
		got, err := toNative(rc)
		observer.EndToNative("toNative", beforeNative, models.Records{rc}, got, err)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := recorder.WriteTo("test_data"); err != nil {
		t.Fatal(err)
	}
}

func strippedGroup(rec client.DNSRecord) client.DNSRecord {
	rec.Group = nil
	return rec
}

func TestClampTTL(t *testing.T) {
	tests := []struct {
		in, want uint32
	}{
		{0, defaultTTL},
		{30, minTTL},
		{60, 60},
		{600, 600},
		{3600, 3600},
		{86400, maxTTL},
	}
	for _, tt := range tests {
		if got := clampTTL(tt.in); got != tt.want {
			t.Errorf("clampTTL(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestApexAliasRejected(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("@", 600, "ALIAS", "target.example.com.")
	errs := AuditRecords(models.Records{rc})
	if len(errs) == 0 {
		t.Fatal("expected apex ALIAS to be rejected")
	}
}

func TestNsAtApexRejected(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("@", 600, "NS", "ns1.example.net.")
	errs := AuditRecords(models.Records{rc})
	if len(errs) == 0 {
		t.Fatal("expected apex NS to be rejected")
	}
}

func TestIsApexNS(t *testing.T) {
	if !isApexNS("example.com", client.DNSRecord{Type: "NS", Name: "@", Nameserver: "launch1.spaceship.net"}) {
		t.Fatal("expected apex Spaceship NS to be skipped")
	}
	if !isApexNS("example.com", client.DNSRecord{Type: "NS", Name: "@", Nameserver: "ns1.example.net"}) {
		t.Fatal("expected apex custom NS to be skipped")
	}
	if isApexNS("example.com", client.DNSRecord{Type: "NS", Name: "sub", Nameserver: "ns1.example.net"}) {
		t.Fatal("non-apex NS should not be skipped")
	}
}

func TestCheckNSModificationsDropsApexNS(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	dc.Records = models.Records{
		dc.MustNewRecordConfig("@", 600, "NS", "launch1.spaceship.net."),
		dc.MustNewRecordConfig("@", 600, "NS", "ns1.example.net."),
		dc.MustNewRecordConfig("www", 600, "A", "192.0.2.1"),
	}
	checkNSModifications(dc)
	if len(dc.Records) != 1 || dc.Records[0].Type != "A" {
		t.Fatalf("records = %v", dc.Records)
	}
}

func TestNameserverUpdateBasicVsCustom(t *testing.T) {
	basic := nameserverUpdate(normalizeHosts(defaultNS))
	if basic.Provider != client.BasicNameserverProvider || len(basic.Hosts) != 0 {
		t.Fatalf("basic request = %+v", basic)
	}
	custom := nameserverUpdate([]string{"ns1.example.net", "ns2.example.net"})
	if custom.Provider != client.CustomNameserverProvider {
		t.Fatalf("custom provider = %q", custom.Provider)
	}
	if strings.Join(custom.Hosts, ",") != "ns1.example.net,ns2.example.net" {
		t.Fatalf("custom hosts = %v", custom.Hosts)
	}
}

func TestToNativePreservesNullMXAndSRVDot(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	mx, err := toNative(dc.MustNewRecordConfig("@", 300, "MX", 0, "."))
	if err != nil {
		t.Fatalf("MX: %v", err)
	}
	if mx.Exchange != "." {
		t.Errorf("MX Exchange = %q, want %q", mx.Exchange, ".")
	}

	srv, err := toNative(dc.MustNewRecordConfig("_sip._tcp", 300, "SRV", 15, 65, 75, "."))
	if err != nil {
		t.Fatalf("SRV: %v", err)
	}
	if srv.Target != "." {
		t.Errorf("SRV Target = %q, want %q", srv.Target, ".")
	}
}

func TestJoinAndSplitPrefixedLabel(t *testing.T) {
	label := joinPrefixedLabel([]string{"_sip", "_tcp"}, "@")
	if label != "_sip._tcp" {
		t.Fatalf("join apex SRV = %q", label)
	}
	prefix, rest, err := splitPrefixedLabel("_sip._tcp.foo", 2)
	if err != nil {
		t.Fatal(err)
	}
	if prefix[0] != "_sip" || prefix[1] != "_tcp" || rest != "foo" {
		t.Fatalf("split = %v %q", prefix, rest)
	}
}

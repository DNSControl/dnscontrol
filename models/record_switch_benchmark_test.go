package models

import (
	"net/netip"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
)

var recordSwitchBenchmarkSink uint64

//go:noinline
func dispatchByTypeNum(rc *RecordConfig) uint64 {
	switch rc.TypeNum {
	case dnsv2.TypeA:
		f := rc.AsA()
		return uint64(dnsv2.TypeA) + uint64(f.Addr.BitLen())
	case dnsv2.TypeAAAA:
		f := rc.AsAAAA()
		return uint64(dnsv2.TypeAAAA) + uint64(f.Addr.BitLen())
	case dnsv2.TypeCNAME:
		f := rc.AsCNAME()
		return uint64(dnsv2.TypeCNAME) + uint64(len(f.Target))
	case dnsv2.TypeMX:
		f := rc.AsMX()
		return uint64(dnsv2.TypeMX) + uint64(f.Preference) + uint64(len(f.Mx))
	case dnsv2.TypeNS:
		f := rc.AsNS()
		return uint64(dnsv2.TypeNS) + uint64(len(f.Ns))
	case dnsv2.TypeSRV:
		f := rc.AsSRV()
		return uint64(dnsv2.TypeSRV) + uint64(f.Priority) + uint64(f.Weight) + uint64(f.Port) + uint64(len(f.Target))
	case dnsv2.TypeTXT:
		f := rc.AsTXT()
		return uint64(dnsv2.TypeTXT) + uint64(len(f.Txt[0]))
	case dnsv2.TypeCAA:
		f := rc.AsCAA()
		return uint64(dnsv2.TypeCAA) + uint64(f.Flag) + uint64(len(f.Tag)) + uint64(len(f.Value))
	default:
		return 0
	}
}

//go:noinline
func dispatchByRDATAType(rc *RecordConfig) uint64 {
	switch f := rc.GetRDATA().(type) {
	case dnsrdatav2.A:
		return uint64(dnsv2.TypeA) + uint64(f.Addr.BitLen())
	case dnsrdatav2.AAAA:
		return uint64(dnsv2.TypeAAAA) + uint64(f.Addr.BitLen())
	case dnsrdatav2.CNAME:
		return uint64(dnsv2.TypeCNAME) + uint64(len(f.Target))
	case dnsrdatav2.MX:
		return uint64(dnsv2.TypeMX) + uint64(f.Preference) + uint64(len(f.Mx))
	case dnsrdatav2.NS:
		return uint64(dnsv2.TypeNS) + uint64(len(f.Ns))
	case dnsrdatav2.SRV:
		return uint64(dnsv2.TypeSRV) + uint64(f.Priority) + uint64(f.Weight) + uint64(f.Port) + uint64(len(f.Target))
	case dnsrdatav2.TXT:
		return uint64(dnsv2.TypeTXT) + uint64(len(f.Txt[0]))
	case dnsrdatav2.CAA:
		return uint64(dnsv2.TypeCAA) + uint64(f.Flag) + uint64(len(f.Tag)) + uint64(len(f.Value))
	default:
		return 0
	}
}

func BenchmarkRecordConfigSwitchDispatch(b *testing.B) {
	records := Records{
		{TypeNum: dnsv2.TypeA, rdata: dnsrdatav2.A{Addr: netip.MustParseAddr("192.0.2.1")}},
		{TypeNum: dnsv2.TypeAAAA, rdata: dnsrdatav2.AAAA{Addr: netip.MustParseAddr("2001:db8::1")}},
		{TypeNum: dnsv2.TypeCNAME, rdata: dnsrdatav2.CNAME{Target: "target.example."}},
		{TypeNum: dnsv2.TypeMX, rdata: dnsrdatav2.MX{Preference: 10, Mx: "mx.example."}},
		{TypeNum: dnsv2.TypeNS, rdata: dnsrdatav2.NS{Ns: "ns.example."}},
		{TypeNum: dnsv2.TypeSRV, rdata: dnsrdatav2.SRV{Priority: 10, Weight: 20, Port: 443, Target: "service.example."}},
		{TypeNum: dnsv2.TypeTXT, rdata: dnsrdatav2.TXT{Txt: []string{"benchmark"}}},
		{TypeNum: dnsv2.TypeCAA, rdata: dnsrdatav2.CAA{Tag: "issue", Value: "ca.example"}},
	}
	defaultRecord := Records{
		{TypeNum: dnsv2.TypePTR, rdata: dnsrdatav2.PTR{}},
	}

	benchmarks := []struct {
		name    string
		records []*RecordConfig
	}{
		{name: "single", records: records[:1]},
		{name: "mixed", records: records},
		{name: "default", records: defaultRecord},
	}

	for _, benchmark := range benchmarks {
		for _, rc := range benchmark.records {
			if got, want := dispatchByRDATAType(rc), dispatchByTypeNum(rc); got != want {
				b.Fatalf("dispatch results differ for type %d: RDATA=%d TypeNum=%d", rc.TypeNum, got, want)
			}
		}

		mask := len(benchmark.records) - 1
		b.Run(benchmark.name+"/TypeNum", func(b *testing.B) {
			b.ReportAllocs()
			var sum uint64
			i := 0
			for b.Loop() {
				sum += dispatchByTypeNum(benchmark.records[i&mask])
				i++
			}
			recordSwitchBenchmarkSink = sum
		})

		b.Run(benchmark.name+"/RDATAType", func(b *testing.B) {
			b.ReportAllocs()
			var sum uint64
			i := 0
			for b.Loop() {
				sum += dispatchByRDATAType(benchmark.records[i&mask])
				i++
			}
			recordSwitchBenchmarkSink = sum
		})
	}
}

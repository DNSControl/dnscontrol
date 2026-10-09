package privatetypes

import (
	"slices"
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
)

func TestRecordCatalog(t *testing.T) {
	for _, name := range []string{"A", "TXT", "NS", "HINFO"} {
		if typ, ok := LookupRecordType(name); !ok || typ.Pseudo {
			t.Errorf("ordinary type %s = %+v, %v", name, typ, ok)
		}
	}
	for _, name := range []string{"ALIAS", "AKAMAITLC", "R53_ALIAS", "BUNNY_DNS_PZ"} {
		if typ, ok := LookupRecordType(name); !ok || !typ.Pseudo {
			t.Errorf("pseudo-type %s = %+v, %v", name, typ, ok)
		}
	}
	for _, name := range []string{"UNRECOGNIZED", "ANY", "AXFR", "IXFR", "OPT", "TKEY", "TSIG", "IMPORT_TRANSFORM"} {
		if _, ok := LookupRecordType(name); ok {
			t.Errorf("non-zone type %s accepted", name)
		}
	}
	if !slices.IsSortedFunc(RecordTypes(), func(a, b RecordType) int {
		return strings.Compare(a.Name, b.Name)
	}) {
		t.Fatal("catalog is not sorted")
	}
}

func TestCatalogClassificationIsExplicit(t *testing.T) {
	// Model an ordinary type installed by the DNS library with both an
	// underscore and a private-use codepoint. Neither makes it a pseudo-type.
	const codepoint = 65499
	const name = "ORDINARY_TEST"
	dnsv2.TypeToRR[codepoint] = func() dnsv2.RR { return &dnsv2.A{} }
	dnsv2.TypeToString[codepoint] = name
	dnsv2.StringToType[name] = codepoint
	t.Cleanup(func() {
		delete(dnsv2.TypeToRR, codepoint)
		delete(dnsv2.TypeToString, codepoint)
		delete(dnsv2.StringToType, name)
		delete(TypeToMakeRDATA, codepoint)
		catalogVersion++
	})
	before := CatalogVersion()
	RegisterMaker(codepoint, nil)
	if typ, ok := LookupRecordType(name); !ok || typ.Pseudo {
		t.Fatalf("classification = %+v, %v", typ, ok)
	}
	if CatalogVersion() == before {
		t.Fatal("catalog registration did not invalidate finalization")
	}
}

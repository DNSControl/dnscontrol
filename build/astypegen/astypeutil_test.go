package main_test

import (
	"testing"

	gen "github.com/DNSControl/dnscontrol/v5/build/astypegen"
)

type sampleRD struct{}

func TestStructName(t *testing.T) {
	// Value: name + this test package's path.
	name, pkgPath := gen.StructName(sampleRD{})
	if name != "sampleRD" {
		t.Errorf("StructName(value): name=%q, want %q", name, "sampleRD")
	}
	if pkgPath == "" {
		t.Errorf("StructName(value): pkgPath is empty, want a non-empty package path")
	}

	// Pointer is dereferenced to the same name/path.
	pName, pPkg := gen.StructName(&sampleRD{})
	if pName != name || pPkg != pkgPath {
		t.Errorf("StructName(pointer) = (%q,%q), want (%q,%q)", pName, pPkg, name, pkgPath)
	}

	// nil yields empties (generator skips these).
	if n, p := gen.StructName(nil); n != "" || p != "" {
		t.Errorf("StructName(nil) = (%q,%q), want empty", n, p)
	}
}

func TestNamespace(t *testing.T) {
	cases := []struct {
		pkgPath   string
		wantAlias string
		wantOK    bool
	}{
		{"codeberg.org/miekg/dns/rdata", "dnsrdatav2", true},
		{"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata", "privatetypesrdata", true},
		// Version-independent: a future major version must still map correctly.
		{"github.com/DNSControl/dnscontrol/v6/pkg/privatetypes/rdata", "privatetypesrdata", true},
		// Anything else is skipped.
		{"codeberg.org/miekg/dns", "", false},
		{"some/other/package", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		alias, ok := gen.Namespace(c.pkgPath)
		if alias != c.wantAlias || ok != c.wantOK {
			t.Errorf("Namespace(%q) = (%q, %v), want (%q, %v)", c.pkgPath, alias, ok, c.wantAlias, c.wantOK)
		}
	}
}

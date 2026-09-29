package rdatafields_test

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	_ "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
	"github.com/DNSControl/dnscontrol/v5/pkg/rdatafields"
)

var update = flag.Bool("update", false, "rewrite testdata/classification.txt")

// TestClassificationGolden pins the Kind of every field of every rtype
// dnscontrol knows about. When the miekg/dns rdata package gains a type or
// changes a tag, this test fails and a human decides whether the new field
// is a hostname. Re-generate with:
//
//	go test ./pkg/rdatafields/ -update
func TestClassificationGolden(t *testing.T) {
	const golden = "testdata/classification.txt"

	got := report()
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("classification changed; review the diff and re-run with -update.\n--- got ---\n%s", got)
	}
}

func report() string {
	seen := map[string]bool{}
	var lines []string
	for _, name := range privatetypes.GetAllTypeNames() {
		newFn, ok := dnsv2.TypeToRR[dnsv2.StringToType[name]]
		if !ok {
			continue
		}
		rd := newFn().Data()
		rt := reflect.TypeOf(rd)
		for rt != nil && rt.Kind() == reflect.Pointer {
			rt = rt.Elem()
		}
		if rt == nil || rt.Kind() != reflect.Struct || seen[rt.String()] {
			continue
		}
		seen[rt.String()] = true

		for _, field := range rdatafields.Fields(rd) {
			f := rt.Field(field.Index)
			lines = append(lines, fmt.Sprintf("%-12s %-20s %-12s dns:%q", rt.Name(), field.Name, field.Kind, f.Tag.Get("dns")))
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}

// TestSpotChecks guards the decisions that the tag rule alone gets wrong.
func TestSpotChecks(t *testing.T) {
	for _, tc := range []struct {
		structName, field, tag string
		want                   rdatafields.Kind
	}{
		{"CNAME", "Target", `dns:"cname"`, rdatafields.KindTargetHost},
		{"MX", "Mx", `dns:"cname"`, rdatafields.KindTargetHost},
		{"SRV", "Target", `dns:"name"`, rdatafields.KindTargetHost},
		{"SOA", "Ns", `dns:"cname"`, rdatafields.KindTargetHost},
		{"SOA", "Mbox", `dns:"mname"`, rdatafields.KindMailbox},
		{"RP", "Txt", `dns:"name"`, rdatafields.KindOpaqueName},
		{"TSIG", "Algorithm", `dns:"name"`, rdatafields.KindOpaqueName},
		{"RRSIG", "SignerName", `dns:"name"`, rdatafields.KindOpaqueName},
		{"DS", "Digest", `dns:"hex"`, rdatafields.KindHexLower},
		{"CAA", "Value", `dns:"any"`, rdatafields.KindOther},
		{"ALIAS", "Target", `dnscontrol:"targethost"`, rdatafields.KindTargetHost},
		{"AZUREALIAS", "Target", ``, rdatafields.KindOther},
		// Obsolete types are skipped even though the tag rule would match.
		{"MB", "Mb", `dns:"cname"`, rdatafields.KindOther},
		{"PX", "Map822", `dns:"name"`, rdatafields.KindOther},
	} {
		f := reflect.StructField{Name: tc.field, Tag: reflect.StructTag(tc.tag), Type: reflect.TypeOf("")}
		if got := rdatafields.Classify(tc.structName, f); got != tc.want {
			t.Errorf("Classify(%s.%s, %s) = %v, want %v", tc.structName, tc.field, tc.tag, got, tc.want)
		}
	}
}

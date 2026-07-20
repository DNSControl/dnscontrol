package diff2

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/fatih/color"
	"github.com/kylelemons/godebug/diff"
)

func init() {
	// Disable colorizing the output.
	// NOTE: "go test ./..." turns it off automatically but "cd
	// pkg/diff2 && go test" does not. Without this statement, the
	// latter fails.
	color.NoColor = true
}

// Stringify the datastructures (for debugging)

func (c Change) String() string {
	var buf bytes.Buffer
	b := &buf

	fmt.Fprintf(b, "Change: verb=%v\n", c.Type)
	fmt.Fprintf(b, "    key=%v\n", c.Key)
	if c.HintOnlyTTL {
		fmt.Fprint(b, "    Hints=OnlyTTL\n", c.Key)
	}
	if len(c.Old) != 0 {
		fmt.Fprintf(b, "    old=%v\n", c.Old)
	}
	if len(c.New) != 0 {
		fmt.Fprintf(b, "    new=%v\n", c.New)
	}
	fmt.Fprintf(b, "    msg=%q\n", c.Msgs)

	return b.String()
}

func (cl ChangeList) String() string {
	var buf bytes.Buffer
	b := &buf

	fmt.Fprintf(b, "ChangeList: len=%d\n", len(cl))
	for i, j := range cl {
		fmt.Fprintf(b, "%02d: %s", i, j)
	}

	return b.String()
}

// Make sample data

func makeRec(label, rtype, content string) *models.RecordConfig {
	origin := "f.com"
	r := models.RecordConfig{TTL: 300}
	r.SetLabel(label, origin)
	if err := r.PopulateFromString(rtype, content, origin); err != nil {
		panic(err)
	}
	return &r
}

func makeRecTTL(label, rtype, content string, ttl uint32) *models.RecordConfig {
	r := makeRec(label, rtype, content)
	r.TTL = ttl
	return r
}

var (
	testDataAA1234       = makeRec("laba", "A", "1.2.3.4")         // [ 0]
	testDataAA5678       = makeRec("laba", "A", "5.6.7.8")         //
	testDataAA1234ttl700 = makeRecTTL("laba", "A", "1.2.3.4", 700) //
	testDataAA5678ttl700 = makeRecTTL("laba", "A", "5.6.7.8", 700) //
	testDataAMX10a       = makeRec("laba", "MX", "10 laba")        // [ 1]
	testDataCCa          = makeRec("labc", "CNAME", "laba")        // [ 2]
	testDataEA15         = makeRec("labe", "A", "10.10.10.15")     // [ 3]
	e4                   = makeRec("labe", "A", "10.10.10.16")     // [ 4]
	e5                   = makeRec("labe", "A", "10.10.10.17")     // [ 5]
	e6                   = makeRec("labe", "A", "10.10.10.18")     // [ 6]
	e7                   = makeRec("labg", "NS", "laba")           // [ 7]
	e8                   = makeRec("labg", "NS", "labb")           // [ 8]
	e9                   = makeRec("labg", "NS", "labc")           // [ 9]
	e10                  = makeRec("labg", "NS", "labe")           // [10]
	e11mx                = makeRec("labh", "MX", "22 ttt")         // [11]
	e11                  = makeRec("labh", "CNAME", "labd")        // [11]
	testDataApexMX1aaa   = makeRec("", "MX", "1 aaa")
)

var (
	testDataAA1234clone = makeRec("laba", "A", "1.2.3.4")     // [ 0']
	testDataAA12345     = makeRec("laba", "A", "1.2.3.5")     // [ 1']
	testDataAMX20b      = makeRec("laba", "MX", "20 labb")    // [ 2']
	d3                  = makeRec("labe", "A", "10.10.10.95") // [ 3']
	d4                  = makeRec("labe", "A", "10.10.10.96") // [ 4']
	d5                  = makeRec("labe", "A", "10.10.10.97") // [ 5']
	d6                  = makeRec("labe", "A", "10.10.10.98") // [ 6']
	d7                  = makeRec("labf", "TXT", "foo")       // [ 7']
	d8                  = makeRec("labg", "NS", "labf")       // [ 8']
	d9                  = makeRec("labg", "NS", "laba")       // [ 9']
	d10                 = makeRec("labg", "NS", "labe")       // [10']
	d11                 = makeRec("labg", "NS", "labb")       // [11']
	d12                 = makeRec("labh", "A", "1.2.3.4")     // [12']
	d13                 = makeRec("labc", "CNAME", "labe")    // [13']
	testDataApexMX22bbb = makeRec("", "MX", "22 bbb")
)

func justMsgString(cl ChangeList) string {
	msgs := justMsgs(cl)
	return strings.Join(msgs, "\n")
}

func compareMsgs(t *testing.T, fnname, testname, testpart string, gotcc ChangeList, wantstring string, wantstringdefault string) {
	wantstring = coalesce(wantstring, wantstringdefault)
	t.Helper()
	gs := strings.TrimSpace(justMsgString(gotcc))
	ws := strings.TrimSpace(wantstring)
	d := diff.Diff(gs, ws)
	if d != "" {
		t.Errorf("%s()/%s (wantMsgs:%s):\n===got===\n%s\n===want===\n%s\n===diff===\n%s\n===", fnname, testname, testpart, gs, ws, d)
	}
}

func compareCL(t *testing.T, fnname, testname, testpart string, gotcl ChangeList, wantstring string) {
	t.Helper()
	gs := strings.TrimSpace(gotcl.String())
	ws := strings.TrimSpace(wantstring)
	d := diff.Diff(gs, ws)
	if d != "" {
		t.Errorf("%s()/%s (wantChange%s):\n===got===\n%s\n===want===\n%s\n===diff===\n%s\n===", fnname, testname, testpart, gs, ws, d)
	}
}

func compareACC(t *testing.T, fnname, testname, testpart string, gotacc int, wantacc int) {
	t.Helper()
	if gotacc != wantacc {
		t.Errorf("%s()/%s (wantChange%s):\n===got===\n%v\n===want===\n%v\n", fnname, testname, testpart, gotacc, wantacc)
	}
}

func Test_analyzeByRecordSet(t *testing.T) {
	type args struct {
		origin            string
		existing, desired models.Records
		compFn            ComparableFunc
	}

	origin := "f.com"

	tests := []struct {
		name            string
		args            args
		wantMsgs        string
		wantChangeRSet  string
		wantMsgsRSet    string
		wantChangeLabel string
		wantMsgsLabel   string
		wantChangeRec   string
		wantMsgsRec     string
		wantChangeZone  string
		wantChangeCount int
	}{
		{
			name: "oneequal",
			args: args{
				origin:   origin,
				existing: models.Records{testDataAA1234},
				desired:  models.Records{testDataAA1234clone},
			},
			wantChangeCount: 0,
			wantMsgs:        "", // Empty
			wantChangeRSet:  "ChangeList: len=0",
			wantChangeLabel: "ChangeList: len=0",
			wantChangeRec:   "ChangeList: len=0",
		},

		{
			name: "onediff",
			args: args{
				origin:   origin,
				existing: models.Records{testDataAA1234, testDataAMX10a},
				desired:  models.Records{testDataAA1234clone, testDataAMX20b},
			},
			wantChangeCount: 1,
			wantMsgs:        "± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)",
			wantChangeRSet: `
ChangeList: len=1
00: Change: verb=CHANGE
    key={laba.f.com MX}
    old=[10 laba.f.com.]
    new=[20 labb.f.com.]
    msg=["± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)"]
`,
			wantChangeLabel: `
ChangeList: len=1
00: Change: verb=CHANGE
    key={laba.f.com }
    old=[1.2.3.4 10 laba.f.com.]
    new=[1.2.3.4 20 labb.f.com.]
    msg=["± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)"]
`,
			wantChangeRec: `
ChangeList: len=1
00: Change: verb=CHANGE
    key={laba.f.com MX}
    old=[10 laba.f.com.]
    new=[20 labb.f.com.]
    msg=["± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)"]
`,
		},

		{
			name: "apex",
			args: args{
				origin:   origin,
				existing: models.Records{testDataAA1234, testDataApexMX1aaa},
				desired:  models.Records{testDataAA1234clone, testDataApexMX22bbb},
			},
			wantChangeCount: 1,
			wantMsgs:        "± MODIFY f.com MX (1 aaa.f.com. ttl=300) -> (22 bbb.f.com. ttl=300)",
			wantChangeRSet: `
ChangeList: len=1
00: Change: verb=CHANGE
    key={f.com MX}
    old=[1 aaa.f.com.]
    new=[22 bbb.f.com.]
    msg=["± MODIFY f.com MX (1 aaa.f.com. ttl=300) -> (22 bbb.f.com. ttl=300)"]
`,
			wantChangeLabel: `
ChangeList: len=1
00: Change: verb=CHANGE
    key={f.com }
    old=[1 aaa.f.com.]
    new=[22 bbb.f.com.]
    msg=["± MODIFY f.com MX (1 aaa.f.com. ttl=300) -> (22 bbb.f.com. ttl=300)"]
`,
			wantChangeRec: `
ChangeList: len=1
00: Change: verb=CHANGE
    key={f.com MX}
    old=[1 aaa.f.com.]
    new=[22 bbb.f.com.]
    msg=["± MODIFY f.com MX (1 aaa.f.com. ttl=300) -> (22 bbb.f.com. ttl=300)"]
`,
		},

		{
			name: "firsta",
			args: args{
				origin:   origin,
				existing: models.Records{testDataAA1234, testDataAMX10a},
				desired:  models.Records{testDataAA1234clone, testDataAA12345, testDataAMX20b},
			},
			wantChangeCount: 2,
			wantMsgs: `
± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)
+ CREATE laba.f.com A 1.2.3.5 ttl=300
`,
			wantMsgsLabel: `
+ CREATE laba.f.com A 1.2.3.5 ttl=300
± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)
`,
			wantChangeRSet: `
ChangeList: len=2
00: Change: verb=CHANGE
    key={laba.f.com MX}
    old=[10 laba.f.com.]
    new=[20 labb.f.com.]
    msg=["± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)"]
01: Change: verb=CHANGE
    key={laba.f.com A}
    old=[1.2.3.4]
    new=[1.2.3.4 1.2.3.5]
    msg=["+ CREATE laba.f.com A 1.2.3.5 ttl=300"]
`,
			wantChangeLabel: `
ChangeList: len=1
00: Change: verb=CHANGE
    key={laba.f.com }
    old=[1.2.3.4 10 laba.f.com.]
    new=[1.2.3.4 1.2.3.5 20 labb.f.com.]
    msg=["+ CREATE laba.f.com A 1.2.3.5 ttl=300" "± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)"]
		`,
			wantChangeRec: `
ChangeList: len=2
00: Change: verb=CHANGE
    key={laba.f.com MX}
    old=[10 laba.f.com.]
    new=[20 labb.f.com.]
    msg=["± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)"]
01: Change: verb=CREATE
    key={laba.f.com A}
    new=[1.2.3.5]
    msg=["+ CREATE laba.f.com A 1.2.3.5 ttl=300"]
`,
		},

		{
			name: "order forward and backward dependent records",
			args: args{
				origin:   origin,
				existing: models.Records{testDataAA1234, testDataCCa},
				desired:  models.Records{d13, d3},
			},
			wantChangeCount: 3,
			wantMsgs: `
+ CREATE labe.f.com A 10.10.10.95 ttl=300
± MODIFY labc.f.com CNAME (laba.f.com. ttl=300) -> (labe.f.com. ttl=300)
- DELETE laba.f.com A 1.2.3.4 ttl=300
		`,
			wantChangeRSet: `
ChangeList: len=3
00: Change: verb=CREATE
    key={labe.f.com A}
    new=[10.10.10.95]
    msg=["+ CREATE labe.f.com A 10.10.10.95 ttl=300"]
01: Change: verb=CHANGE
    key={labc.f.com CNAME}
    old=[laba.f.com.]
    new=[labe.f.com.]
    msg=["± MODIFY labc.f.com CNAME (laba.f.com. ttl=300) -> (labe.f.com. ttl=300)"]
02: Change: verb=DELETE
    key={laba.f.com A}
    old=[1.2.3.4]
    msg=["- DELETE laba.f.com A 1.2.3.4 ttl=300"]
		`,
			wantChangeLabel: `
ChangeList: len=3
00: Change: verb=CREATE
    key={labe.f.com }
    new=[10.10.10.95]
    msg=["+ CREATE labe.f.com A 10.10.10.95 ttl=300"]
01: Change: verb=CHANGE
    key={labc.f.com }
    old=[laba.f.com.]
    new=[labe.f.com.]
    msg=["± MODIFY labc.f.com CNAME (laba.f.com. ttl=300) -> (labe.f.com. ttl=300)"]
02: Change: verb=DELETE
    key={laba.f.com }
    old=[1.2.3.4]
    msg=["- DELETE laba.f.com A 1.2.3.4 ttl=300"]
`,
			wantChangeRec: `
ChangeList: len=3
00: Change: verb=CREATE
    key={labe.f.com A}
    new=[10.10.10.95]
    msg=["+ CREATE labe.f.com A 10.10.10.95 ttl=300"]
01: Change: verb=CHANGE
    key={labc.f.com CNAME}
    old=[laba.f.com.]
    new=[labe.f.com.]
    msg=["± MODIFY labc.f.com CNAME (laba.f.com. ttl=300) -> (labe.f.com. ttl=300)"]
02: Change: verb=DELETE
    key={laba.f.com A}
    old=[1.2.3.4]
    msg=["- DELETE laba.f.com A 1.2.3.4 ttl=300"]
`,
		},

		{
			name: "big",
			args: args{
				origin:   origin,
				existing: models.Records{testDataAA1234, testDataAMX10a, testDataCCa, testDataEA15, e4, e5, e6, e7, e8, e9, e10, e11},
				desired:  models.Records{testDataAA1234clone, testDataAA12345, testDataAMX20b, d3, d4, d5, d6, d7, d8, d9, d10, d11, d12},
			},
			wantChangeCount: 11,
			wantMsgs: `
+ CREATE labf.f.com TXT "foo" ttl=300
± MODIFY labg.f.com NS (labc.f.com. ttl=300) -> (labf.f.com. ttl=300)
- DELETE labh.f.com CNAME labd.f.com. ttl=300
+ CREATE labh.f.com A 1.2.3.4 ttl=300
- DELETE labc.f.com CNAME laba.f.com. ttl=300
± MODIFY labe.f.com A (10.10.10.15 ttl=300) -> (10.10.10.95 ttl=300)
± MODIFY labe.f.com A (10.10.10.16 ttl=300) -> (10.10.10.96 ttl=300)
± MODIFY labe.f.com A (10.10.10.17 ttl=300) -> (10.10.10.97 ttl=300)
± MODIFY labe.f.com A (10.10.10.18 ttl=300) -> (10.10.10.98 ttl=300)
± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)
+ CREATE laba.f.com A 1.2.3.5 ttl=300
		`,
			wantChangeRSet: `
ChangeList: len=8
00: Change: verb=CREATE
    key={labf.f.com TXT}
    new=["foo"]
    msg=["+ CREATE labf.f.com TXT \"foo\" ttl=300"]
01: Change: verb=CHANGE
    key={labg.f.com NS}
    old=[laba.f.com. labb.f.com. labc.f.com. labe.f.com.]
    new=[laba.f.com. labb.f.com. labe.f.com. labf.f.com.]
    msg=["± MODIFY labg.f.com NS (labc.f.com. ttl=300) -> (labf.f.com. ttl=300)"]
02: Change: verb=DELETE
    key={labh.f.com CNAME}
    old=[labd.f.com.]
    msg=["- DELETE labh.f.com CNAME labd.f.com. ttl=300"]
03: Change: verb=CREATE
    key={labh.f.com A}
    new=[1.2.3.4]
    msg=["+ CREATE labh.f.com A 1.2.3.4 ttl=300"]
04: Change: verb=DELETE
    key={labc.f.com CNAME}
    old=[laba.f.com.]
    msg=["- DELETE labc.f.com CNAME laba.f.com. ttl=300"]
05: Change: verb=CHANGE
    key={labe.f.com A}
    old=[10.10.10.15 10.10.10.16 10.10.10.17 10.10.10.18]
    new=[10.10.10.95 10.10.10.96 10.10.10.97 10.10.10.98]
    msg=["± MODIFY labe.f.com A (10.10.10.15 ttl=300) -> (10.10.10.95 ttl=300)" "± MODIFY labe.f.com A (10.10.10.16 ttl=300) -> (10.10.10.96 ttl=300)" "± MODIFY labe.f.com A (10.10.10.17 ttl=300) -> (10.10.10.97 ttl=300)" "± MODIFY labe.f.com A (10.10.10.18 ttl=300) -> (10.10.10.98 ttl=300)"]
06: Change: verb=CHANGE
    key={laba.f.com MX}
    old=[10 laba.f.com.]
    new=[20 labb.f.com.]
    msg=["± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)"]
07: Change: verb=CHANGE
    key={laba.f.com A}
    old=[1.2.3.4]
    new=[1.2.3.4 1.2.3.5]
    msg=["+ CREATE laba.f.com A 1.2.3.5 ttl=300"]
	`,
			wantMsgsLabel: `
+ CREATE labf.f.com TXT "foo" ttl=300
± MODIFY labg.f.com NS (labc.f.com. ttl=300) -> (labf.f.com. ttl=300)
- DELETE labh.f.com CNAME labd.f.com. ttl=300
+ CREATE labh.f.com A 1.2.3.4 ttl=300
- DELETE labc.f.com CNAME laba.f.com. ttl=300
± MODIFY labe.f.com A (10.10.10.15 ttl=300) -> (10.10.10.95 ttl=300)
± MODIFY labe.f.com A (10.10.10.16 ttl=300) -> (10.10.10.96 ttl=300)
± MODIFY labe.f.com A (10.10.10.17 ttl=300) -> (10.10.10.97 ttl=300)
± MODIFY labe.f.com A (10.10.10.18 ttl=300) -> (10.10.10.98 ttl=300)
+ CREATE laba.f.com A 1.2.3.5 ttl=300
± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)
			`,
			wantChangeLabel: `
ChangeList: len=6
00: Change: verb=CREATE
    key={labf.f.com }
    new=["foo"]
    msg=["+ CREATE labf.f.com TXT \"foo\" ttl=300"]
01: Change: verb=CHANGE
    key={labg.f.com }
    old=[laba.f.com. labb.f.com. labc.f.com. labe.f.com.]
    new=[laba.f.com. labb.f.com. labe.f.com. labf.f.com.]
    msg=["± MODIFY labg.f.com NS (labc.f.com. ttl=300) -> (labf.f.com. ttl=300)"]
02: Change: verb=CHANGE
    key={labh.f.com }
    old=[labd.f.com.]
    new=[1.2.3.4]
    msg=["- DELETE labh.f.com CNAME labd.f.com. ttl=300" "+ CREATE labh.f.com A 1.2.3.4 ttl=300"]
03: Change: verb=DELETE
    key={labc.f.com }
    old=[laba.f.com.]
    msg=["- DELETE labc.f.com CNAME laba.f.com. ttl=300"]
04: Change: verb=CHANGE
    key={labe.f.com }
    old=[10.10.10.15 10.10.10.16 10.10.10.17 10.10.10.18]
    new=[10.10.10.95 10.10.10.96 10.10.10.97 10.10.10.98]
    msg=["± MODIFY labe.f.com A (10.10.10.15 ttl=300) -> (10.10.10.95 ttl=300)" "± MODIFY labe.f.com A (10.10.10.16 ttl=300) -> (10.10.10.96 ttl=300)" "± MODIFY labe.f.com A (10.10.10.17 ttl=300) -> (10.10.10.97 ttl=300)" "± MODIFY labe.f.com A (10.10.10.18 ttl=300) -> (10.10.10.98 ttl=300)"]
05: Change: verb=CHANGE
    key={laba.f.com }
    old=[1.2.3.4 10 laba.f.com.]
    new=[1.2.3.4 1.2.3.5 20 labb.f.com.]
    msg=["+ CREATE laba.f.com A 1.2.3.5 ttl=300" "± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)"]
        `,
			wantMsgsRec: `
± MODIFY labe.f.com A (10.10.10.15 ttl=300) -> (10.10.10.95 ttl=300)
± MODIFY labe.f.com A (10.10.10.16 ttl=300) -> (10.10.10.96 ttl=300)
± MODIFY labe.f.com A (10.10.10.17 ttl=300) -> (10.10.10.97 ttl=300)
± MODIFY labe.f.com A (10.10.10.18 ttl=300) -> (10.10.10.98 ttl=300)
+ CREATE labf.f.com TXT "foo" ttl=300
± MODIFY labg.f.com NS (labc.f.com. ttl=300) -> (labf.f.com. ttl=300)
- DELETE labh.f.com CNAME labd.f.com. ttl=300
+ CREATE labh.f.com A 1.2.3.4 ttl=300
- DELETE labc.f.com CNAME laba.f.com. ttl=300
± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)
+ CREATE laba.f.com A 1.2.3.5 ttl=300
		`,
			wantChangeRec: `
ChangeList: len=11
00: Change: verb=CHANGE
    key={labe.f.com A}
    old=[10.10.10.15]
    new=[10.10.10.95]
    msg=["± MODIFY labe.f.com A (10.10.10.15 ttl=300) -> (10.10.10.95 ttl=300)"]
01: Change: verb=CHANGE
    key={labe.f.com A}
    old=[10.10.10.16]
    new=[10.10.10.96]
    msg=["± MODIFY labe.f.com A (10.10.10.16 ttl=300) -> (10.10.10.96 ttl=300)"]
02: Change: verb=CHANGE
    key={labe.f.com A}
    old=[10.10.10.17]
    new=[10.10.10.97]
    msg=["± MODIFY labe.f.com A (10.10.10.17 ttl=300) -> (10.10.10.97 ttl=300)"]
03: Change: verb=CHANGE
    key={labe.f.com A}
    old=[10.10.10.18]
    new=[10.10.10.98]
    msg=["± MODIFY labe.f.com A (10.10.10.18 ttl=300) -> (10.10.10.98 ttl=300)"]
04: Change: verb=CREATE
    key={labf.f.com TXT}
    new=["foo"]
    msg=["+ CREATE labf.f.com TXT \"foo\" ttl=300"]
05: Change: verb=CHANGE
    key={labg.f.com NS}
    old=[labc.f.com.]
    new=[labf.f.com.]
    msg=["± MODIFY labg.f.com NS (labc.f.com. ttl=300) -> (labf.f.com. ttl=300)"]
06: Change: verb=DELETE
    key={labh.f.com CNAME}
    old=[labd.f.com.]
    msg=["- DELETE labh.f.com CNAME labd.f.com. ttl=300"]
07: Change: verb=CREATE
    key={labh.f.com A}
    new=[1.2.3.4]
    msg=["+ CREATE labh.f.com A 1.2.3.4 ttl=300"]
08: Change: verb=DELETE
    key={labc.f.com CNAME}
    old=[laba.f.com.]
    msg=["- DELETE labc.f.com CNAME laba.f.com. ttl=300"]
09: Change: verb=CHANGE
    key={laba.f.com MX}
    old=[10 laba.f.com.]
    new=[20 labb.f.com.]
    msg=["± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)"]
10: Change: verb=CREATE
    key={laba.f.com A}
    new=[1.2.3.5]
    msg=["+ CREATE laba.f.com A 1.2.3.5 ttl=300"]
`,
		},
	}
	for _, tt := range tests {
		models.CanonicalizeTargets(tt.args.existing, tt.args.origin)
		models.CanonicalizeTargets(tt.args.desired, tt.args.origin)

		// Each "analyze*()" should return the same msgs, but a different ChangeList.
		// Sadly the analyze*() functions are destructive to the CompareConfig struct.
		// Therefore we have to run NewCompareConfig() each time.

		t.Run(tt.name, func(t *testing.T) {
			cl, actualChangeCount := analyzeByRecordSet(NewCompareConfig(tt.args.origin, tt.args.existing, tt.args.desired, tt.args.compFn))
			compareMsgs(t, "analyzeByRecordSet", tt.name, "RSet", cl, tt.wantMsgsRSet, tt.wantMsgs)
			compareCL(t, "analyzeByRecordSet", tt.name, "RSet", cl, tt.wantChangeRSet)
			compareACC(t, "analyzeByRecordSet", tt.name, "ACC", actualChangeCount, tt.wantChangeCount)
		})

		t.Run(tt.name, func(t *testing.T) {
			cl, actualChangeCount := analyzeByLabel(NewCompareConfig(tt.args.origin, tt.args.existing, tt.args.desired, tt.args.compFn))
			compareMsgs(t, "analyzeByLabel", tt.name, "Label", cl, tt.wantMsgsLabel, tt.wantMsgs)
			compareCL(t, "analyzeByLabel", tt.name, "Label", cl, tt.wantChangeLabel)
			compareACC(t, "analyzeByLabel", tt.name, "ACC", actualChangeCount, tt.wantChangeCount)
		})

		t.Run(tt.name, func(t *testing.T) {
			cl, actualChangeCount := analyzeByRecord(NewCompareConfig(tt.args.origin, tt.args.existing, tt.args.desired, tt.args.compFn))
			compareMsgs(t, "analyzeByRecord", tt.name, "Rec", cl, tt.wantMsgsRec, tt.wantMsgs)
			compareCL(t, "analyzeByRecord", tt.name, "Rec", cl, tt.wantChangeRec)
			compareACC(t, "analyzeByRecord", tt.name, "ACC", actualChangeCount, tt.wantChangeCount)
		})

		// NB(tlim): There is no analyzeByZone().  diff2.ByZone() uses analyzeByRecord().
	}
}

func coalesce(a string, b string) string {
	if a != "" {
		return a
	}
	return b
}

func mkTargetConfig(x ...*models.RecordConfig) []targetConfig {
	var tc []targetConfig

	models.CanonicalizeTargets(x, "f.com")
	for _, r := range x {
		ct, cf := mkCompareBlobs(r, nil)
		tc = append(tc, targetConfig{
			comparableNoTTL: ct,
			comparableFull:  cf,
			rec:             r,
		})
	}
	return tc
}

func mkTargetConfigMap(x ...*models.RecordConfig) map[string]*targetConfig {
	m := map[string]*targetConfig{}
	for _, v := range mkTargetConfig(x...) {
		m[v.comparableFull] = &v
	}
	return m
}

func Test_diffTargets(t *testing.T) {
	type args struct {
		existing []targetConfig
		desired  []targetConfig
	}
	tests := []struct {
		name string
		args args
		want ChangeList
	}{
		{
			name: "add1changettl",
			args: args{
				existing: mkTargetConfig(testDataAA5678),
				desired:  mkTargetConfig(testDataAA5678ttl700, testDataAA1234ttl700),
			},
			want: ChangeList{
				Change{
					Type: CHANGE,
					Key:  models.RecordKey{NameFQDN: "laba.f.com", Type: "A"},
					New:  models.Records{testDataAA5678ttl700, testDataAA1234ttl700},
					Msgs: []string{
						"± MODIFY-TTL laba.f.com A ttl=(300->700) 5.6.7.8",
						"+ CREATE laba.f.com A 1.2.3.4 ttl=700",
					},
				},
			},
		},

		{
			name: "single",
			args: args{
				existing: mkTargetConfig(testDataAA1234),
				desired:  mkTargetConfig(testDataAA1234),
			},
			// want: ,
		},

		{
			name: "add1",
			args: args{
				existing: mkTargetConfig(testDataAA1234),
				desired:  mkTargetConfig(testDataAA1234, testDataAMX10a),
			},
			want: ChangeList{
				Change{
					Type: CREATE,
					Key:  models.RecordKey{NameFQDN: "laba.f.com", Type: "MX"},
					New:  models.Records{makeRec("laba", "MX", "10 laba.f.com.")},
					Msgs: []string{"+ CREATE laba.f.com MX 10 laba.f.com. ttl=300"},
				},
			},
		},

		{
			name: "del1",
			args: args{
				existing: mkTargetConfig(testDataAA1234, testDataAMX10a),
				desired:  mkTargetConfig(testDataAA1234),
			},
			want: ChangeList{
				Change{
					Type: DELETE,
					Key:  models.RecordKey{NameFQDN: "laba.f.com", Type: "MX"},
					Old:  models.Records{makeRec("laba", "MX", "10 laba.f.com.")},
					Msgs: []string{"- DELETE laba.f.com MX 10 laba.f.com. ttl=300"},
				},
			},
		},

		{
			name: "change2nd",
			args: args{
				existing: mkTargetConfig(testDataAA1234, testDataAMX10a),
				desired:  mkTargetConfig(testDataAA1234, testDataAMX20b),
			},
			want: ChangeList{
				Change{
					Type: CHANGE,
					Key:  models.RecordKey{NameFQDN: "laba.f.com", Type: "MX"},
					Old:  models.Records{testDataAMX10a},
					New:  models.Records{testDataAMX20b},
					Msgs: []string{"± MODIFY laba.f.com MX (10 laba.f.com. ttl=300) -> (20 labb.f.com. ttl=300)"},
				},
			},
		},

		{
			name: "del2nd",
			args: args{
				existing: mkTargetConfig(testDataAA1234, testDataAA5678),
				desired:  mkTargetConfig(testDataAA1234),
			},
			want: ChangeList{
				Change{
					Type: CHANGE,
					Key:  models.RecordKey{NameFQDN: "laba.f.com", Type: "A"},
					Old:  models.Records{testDataAA1234, testDataAA5678},
					New:  models.Records{testDataAA1234},
					Msgs: []string{"- DELETE laba.f.com A 5.6.7.8 ttl=300"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// fmt.Printf("DEBUG: Test %02d\n", i)
			got := diffTargets(tt.args.existing, tt.args.desired)
			g := strings.TrimSpace(justMsgString(got))
			w := strings.TrimSpace(justMsgString(tt.want))
			d := diff.Diff(g, w)
			if d != "" {
				// fmt.Printf("DEBUG: fail %q %q\n", g, w)
				t.Errorf("diffTargets()\n diff=%s", d)
			}
		})
	}
}

func Test_removeCommon(t *testing.T) {
	type args struct {
		existing []targetConfig
		desired  []targetConfig
	}
	tests := []struct {
		name  string
		args  args
		want  []targetConfig
		want1 []targetConfig
	}{
		{
			name: "same",
			args: args{
				existing: mkTargetConfig(testDataAA1234clone),
				desired:  mkTargetConfig(testDataAA1234clone),
			},
			want:  []targetConfig{},
			want1: []targetConfig{},
		},

		{
			name: "disjoint",
			args: args{
				existing: mkTargetConfig(testDataAA1234),
				desired:  mkTargetConfig(testDataAA5678),
			},
			want:  mkTargetConfig(testDataAA1234),
			want1: mkTargetConfig(testDataAA5678),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := removeCommon(tt.args.existing, tt.args.desired)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("removeCommon() got  = %v, want %v", got, tt.want)
			}
			if (got1 != nil || tt.want1 != nil) && !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("removeCommon() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func comparables(s []targetConfig) []string {
	var r []string
	for _, j := range s {
		r = append(r, j.comparableFull)
	}
	return r
}

func Test_filterBy(t *testing.T) {
	type args struct {
		s []targetConfig
		m map[string]*targetConfig
	}
	tests := []struct {
		name string
		args args
		want []targetConfig
	}{
		{
			name: "removeall",
			args: args{
				s: mkTargetConfig(testDataAA1234, testDataAMX10a),
				m: mkTargetConfigMap(testDataAA1234, testDataAMX10a),
			},
			want: []targetConfig{},
		},

		{
			name: "keepall",
			args: args{
				s: mkTargetConfig(testDataAA1234, testDataAMX10a),
				m: mkTargetConfigMap(),
			},
			want: mkTargetConfig(testDataAA1234, testDataAMX10a),
		},

		{
			name: "keepsome",
			args: args{
				s: mkTargetConfig(testDataAA1234, testDataAMX10a),
				m: mkTargetConfigMap(testDataAMX10a),
			},
			want: mkTargetConfig(testDataAA1234),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterBy(tt.args.s, tt.args.m); !reflect.DeepEqual(comparables(got), comparables(tt.want)) {
				t.Errorf("filterBy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_splitTTLOnly(t *testing.T) {
	type args struct {
		existing []targetConfig
		desired  []targetConfig
	}
	tests := []struct {
		name           string
		args           args
		wantExistDiff  []targetConfig
		wantDesireDiff []targetConfig
		wantChanges    string
	}{
		{
			name: "simple",
			args: args{
				existing: mkTargetConfig(testDataAA1234),
				desired:  mkTargetConfig(testDataAA1234ttl700),
			},
			wantExistDiff:  nil,
			wantDesireDiff: nil,
			wantChanges:    "ChangeList: len=1\n00: Change: verb=CHANGE\n    key={laba.f.com A}\n    Hints=OnlyTTL\n{laba.f.com A}    old=[1.2.3.4]\n    new=[1.2.3.4]\n    msg=[\"± MODIFY-TTL laba.f.com A ttl=(300->700) 1.2.3.4\"]\n",
		},

		{
			name: "both",
			args: args{
				existing: mkTargetConfig(testDataAA1234),
				desired:  mkTargetConfig(testDataAA5678),
			},
			wantExistDiff:  mkTargetConfig(testDataAA1234),
			wantDesireDiff: mkTargetConfig(testDataAA5678),
			wantChanges:    "ChangeList: len=0\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExistDiff, gotDesireDiff, gotChanges := findTTLChanges(tt.args.existing, tt.args.desired)
			if !reflect.DeepEqual(gotExistDiff, tt.wantExistDiff) {
				t.Errorf("splitTTLOnly() gotExistDiff = %v, want %v", gotExistDiff, tt.wantExistDiff)
			}
			if !reflect.DeepEqual(gotDesireDiff, tt.wantDesireDiff) {
				t.Errorf("splitTTLOnly() gotDesireDiff = %v, want %v", gotDesireDiff, tt.wantDesireDiff)
			}
			gotChangesString := gotChanges.String()
			if gotChangesString != tt.wantChanges {
				t.Errorf("splitTTLOnly() gotChanges=\n%q, want=\n%q", gotChangesString, tt.wantChanges)
			}
		})
	}
}

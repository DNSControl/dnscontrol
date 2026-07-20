package bind

import (
	"reflect"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
)

func Test_makeFileName(t *testing.T) {
	fmtDefault := "%U.zone"
	fmtBasic := "%U - %T - %D"
	fmtFancy := "%T%?_%D.zone" // Include the tag_ only if there is a tag
	fmtErrorPct := "literal%"
	fmtErrorOpt := "literal%?"
	fmtErrorUnk := "literal%o" // Unknown % verb

	ff := domaintags.DomainNameVarieties{
		NameRaw:     "domy",
		NameASCII:   "idn",
		NameUnicode: "uni",
		UniqueName:  "unique!taga",
		Tag:         "tagy",
	}
	tagless := domaintags.DomainNameVarieties{
		NameRaw:     "domy",
		NameASCII:   "idn",
		NameUnicode: "uni",
		UniqueName:  "unique",
		Tag:         "",
	}

	type args struct {
		format string
		ff     domaintags.DomainNameVarieties
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// Test corner cases and common cases.
		{"literal", args{"literal", ff}, "literal"},
		{"basic", args{fmtBasic, ff}, "domy!tagy - tagy - domy"},
		{"solo", args{"%D", ff}, "domy"},
		{"front", args{"%Daaa", ff}, "domyaaa"},
		{"tail", args{"bbb%D", ff}, "bbbdomy"},
		{"def", args{fmtDefault, ff}, "domy!tagy.zone"},
		{"fanWI", args{fmtFancy, ff}, "tagy_domy.zone"},
		{"fanWO", args{fmtFancy, tagless}, "domy.zone"},
		{"errP", args{fmtErrorPct, ff}, "literal%(format may not end in %)"},
		{"errQ", args{fmtErrorOpt, ff}, "literal%(format may not end in %?)"},
		{"errU", args{fmtErrorUnk, ff}, "literal%(unknown %verb %o)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeFileName(tt.args.format, tt.args.ff); got != tt.want {
				t.Errorf("makeFileName(%q) = %q, want %q", tt.args.format, got, tt.want)
			}
		})
	}
}

func Test_makeFileName_2(t *testing.T) {
	ff1 := *domaintags.MakeDomainNameVarieties(`EXAMple.com`)
	ff2 := *domaintags.MakeDomainNameVarieties(`EXAMple.com!myTag`)
	ff3 := *domaintags.MakeDomainNameVarieties(`рф.com!myTag`)

	tests := []struct {
		name   string
		format string
		want1  string
		want2  string
		want3  string
		descr  string // Not used in test, just for documentation generation
	}{
		// NOTE: "Domain" in these descriptions means the domain name without any split horizon tag. Technically the "Zone".
		{`T`, `%T`, ``, `myTag`, `myTag`, `the tag`},
		{`c`, `%c`, `example.com`, `example.com!myTag`, `xn--p1ai.com!myTag`, `canonical name, globally unique and comparable`},
		{`a`, `%a`, `example.com`, `example.com`, `xn--p1ai.com`, `ASCII domain (Punycode, downcased)`},
		{`u`, `%u`, `example.com`, `example.com`, `рф.com`, `Unicode domain (non-Unicode parts downcased)`},
		{`r`, `%r`, `EXAMple.com`, `EXAMple.com`, `рф.com`, "Raw (unmodified) Domain from `D()` (risky!)"},
		{`f`, `%f`, `example.com`, `example.com_myTag`, `xn--p1ai.com_myTag`, "like `%c` but better for filenames (`%a%?_%T`)"},
		{`F`, `%F`, `example.com`, `myTag_example.com`, `myTag_xn--p1ai.com`, "like `%f` but reversed order (`%T%?_%a`)"},
		{`%?x`, `%?x`, ``, `x`, `x`, "returns `x` if tag exists, otherwise \"\""},
		{`%`, `%%`, `%`, `%`, `%`, `a literal percent sign`},

		// Pre-v4.28 names kept for compatibility (note: pre v4.28 did not permit mixed case domain names, we downcased them here for the tests)
		{`U`, `%U`, `example.com`, `example.com!myTag`, `рф.com!myTag`, "(deprecated, use `%c`) Same as `%D%?!%T` (risky!)"},
		{`D`, `%D`, `example.com`, `example.com`, `рф.com`, "(deprecated, use `%r`) mangles Unicode (risky!)"},
		{`%T%?_%D.zone`, `%T%?_%D.zone`, `example.com.zone`, `myTag_example.com.zone`, `myTag_рф.com.zone`, `mentioned in the docs`},
		{`db_%T%?_%D`, `db_%T%?_%D`, `db_example.com`, `db_myTag_example.com`, `db_myTag_рф.com`, `mentioned in the docs`},
		{`db_%D`, `db_%D`, `db_example.com`, `db_example.com`, `db_рф.com`, `mentioned in the docs`},

		// Examples used in the documentation for the BIND provider
		{`%c.zone`, `%c.zone`, "example.com.zone", "example.com!myTag.zone", "xn--p1ai.com!myTag.zone", "Default format (v4.28 and later)"},
		{`%U.zone`, `%U.zone`, `example.com.zone`, `example.com!myTag.zone`, `рф.com!myTag.zone`, "Default format (pre-v4.28) (risky!)"},
		{`db_%f`, `db_%f`, `db_example.com`, `db_example.com_myTag`, `db_xn--p1ai.com_myTag`, "Recommended in a popular DNS book"},
		{`db_a%?_%T`, `db_%a%?_%T`, `db_example.com`, `db_example.com_myTag`, `db_xn--p1ai.com_myTag`, "same as above but using `%?_`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got1 := makeFileName(tt.format, ff1)
			if got1 != tt.want1 {
				t.Errorf("makeFileName(%q) = ff1 %q, want %q", tt.format, got1, tt.want1)
			}
			got2 := makeFileName(tt.format, ff2)
			if got2 != tt.want2 {
				t.Errorf("makeFileName(%q) = ff2 %q, want %q", tt.format, got2, tt.want2)
			}
			got3 := makeFileName(tt.format, ff3)
			if got3 != tt.want3 {
				t.Errorf("makeFileName(%q) = ff3 %q, want %q", tt.format, got3, tt.want3)
			}
			//Uncomment to regenerate lines used in documentation/provider/bind.md 's table:
			// fmt.Print(strings.ReplaceAll(fmt.Sprintf("| `%s` | %s | `%s` | `%s` | `%s` |\n", tt.format, tt.descr, got1, got2, got3), "``", "`\"\"` (null)"))
			//Uncomment to regenerate the above test cases:
			// fmt.Printf("{`%s`, `%s`, `%s`, `%s`, `%s`, %q},\n", tt.name, tt.format, got1, got2, got3, tt.descr)
		})
	}
}

func Test_makeExtractor(t *testing.T) {
	type args struct {
		format string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"c", args{"%c.zone"}, `(.*)!.+\.zone|(.*)\.zone`, false},
		{"a", args{"%a.zone"}, `(.*)\.zone`, false},
		{"u", args{"%u.zone"}, `(.*)\.zone`, false},
		{"r", args{"%r.zone"}, `(.*)\.zone`, false},
		// TODO: Add test cases.
		{"u", args{"%U.zone"}, `(.*)!.+\.zone|(.*)\.zone`, false},
		{"d", args{"%D.zone"}, `(.*)\.zone`, false},
		{"basic", args{"%U - %T - %D"}, `(.*)!.+ - .* - (.*)|(.*) -  - (.*)`, false},
		{"bk1", args{"db_%U"}, `db_(.*)!.+|db_(.*)`, false},
		{"bk2", args{"db_%T_%D"}, `db_.*_(.*)`, false},
		{"fan", args{"%T%?_%D.zone"}, `.*_(.*)\.zone|(.*)\.zone`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeExtractor(tt.args.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("makeExtractor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("makeExtractor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_extractZonesFromFilenames(t *testing.T) {
	type args struct {
		format string
		names  []string
	}

	// A list of filenames one might find in a directory.
	filelist := []string{
		"foo!one.zone",                // u
		"dom.tld!two.zone",            // u
		"foo.zone",                    // d
		"dom.tld.zone",                // d
		"foo!one - one - foo",         // basic
		"dom.tld!two - two - dom.tld", // basic
		"db_foo",                      // bk1
		"db_dom.tld",                  // bk1
		"db_dom.tld!tag",              // bk1
		"db_inside_foo",               // bk2
		"db_outside_dom.tld",          // bk2
		"db__example.com",             // bk2
		"dom.zone",                    // fan
		"example.com.zone",            // fan (no tag)
		"mytag_example.com.zone",      // fan (w/ tag)
		"dom.zone",                    // fan (no tag)
		"mytag_dom.zone",              // fan (w/ tag)
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{"0", args{"%D.zone", []string{"foo.zone", "dom.tld.zone"}}, []string{"foo", "dom.tld"}},
		{"1", args{"%U.zone", []string{"foo.zone", "dom.tld.zone"}}, []string{"foo", "dom.tld"}},
		{"2", args{"%T%?_%D.zone", []string{"inside_ex.tld.zone", "foo.zone", "dom.tld.zone"}}, []string{"ex.tld", "foo", "dom.tld"}},
		{"d", args{"%D.zone", filelist}, []string{"foo!one", "dom.tld!two", "foo", "dom.tld", "dom", "example.com", "mytag_example.com", "dom", "mytag_dom"}},
		{"u", args{"%U.zone", filelist}, []string{"foo", "dom.tld", "foo", "dom.tld", "dom", "example.com", "mytag_example.com", "dom", "mytag_dom"}},
		{"bk1", args{"db_%U", filelist}, []string{"foo", "dom.tld", "dom.tld", "inside_foo", "outside_dom.tld", "_example.com"}},
		{"bk2", args{"db_%T_%D", filelist}, []string{"foo", "dom.tld", "example.com"}},
		{"fan", args{"%T%?_%D.zone", filelist}, []string{"foo!one", "dom.tld!two", "foo", "dom.tld", "dom", "example.com", "example.com", "dom", "dom"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractZonesFromFilenames(tt.args.format, tt.args.names); !reflect.DeepEqual(got, tt.want) {
				ext, _ := makeExtractor(tt.args.format)
				t.Errorf("extractZonesFromFilenames() = %v, want %v Fm=%s Ex=%s", got, tt.want, tt.args.format, ext)
			}
		})
	}
}

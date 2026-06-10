//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TypeDef represents a single type definition from the YAML
type TypeDef struct {
	Name      string        `yaml:"name"`
	Codepoint int           `yaml:"codepoint"`
	Fields    []FieldDef    `yaml:"fields"`
	TestData  []TestDataDef `yaml:"test_data"`
}

// FieldDef represents a field within a type
type FieldDef struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// TestDataDef represents test data for a type
type TestDataDef struct {
	Name   string                 `yaml:"name"`
	Values map[string]interface{} `yaml:"values"`
}

// Config represents the YAML file structure
type Config struct {
	Types []TypeDef `yaml:"types"`
}

// TypeInfo describes how a mustbe type maps to Go code.
type TypeInfo struct {
	GoType      string // Go type of the struct field
	NeedsOrigin bool   // mustbe call needs an origin string as first arg
	NeedsNetip  bool   // requires "net/netip" import
	IsString    bool   // field is a Go string (affects len/string formatting)
}

var typeInfo = map[string]TypeInfo{
	"RawString":  {GoType: "string", IsString: true},
	"TargetHost": {GoType: "string", IsString: true, NeedsOrigin: true},
	"BoolString": {GoType: "string", IsString: true},
	"Bool":       {GoType: "bool"},
	"Uint8":      {GoType: "uint8"},
	"Uint16":     {GoType: "uint16"},
	"Uint32":     {GoType: "uint32"},
	"IPv4":       {GoType: "netip.Addr", NeedsNetip: true},
	"IPv6":       {GoType: "netip.Addr", NeedsNetip: true},
}

func info(typeName string) TypeInfo {
	ti, ok := typeInfo[typeName]
	if !ok {
		log.Fatalf("unknown mustbe type %q", typeName)
	}
	return ti
}

func anyNeedsNetip(fields []FieldDef) bool {
	for _, f := range fields {
		if info(f.Type).NeedsNetip {
			return true
		}
	}
	return false
}

func anyNonString(fields []FieldDef) bool {
	for _, f := range fields {
		if !info(f.Type).IsString {
			return true
		}
	}
	return false
}

// fieldStringExpr returns a Go expression that converts the named field to a string for printing.
func fieldStringExpr(receiver string, f FieldDef) string {
	ti := info(f.Type)
	switch {
	case ti.IsString:
		return fmt.Sprintf("%s.%s", receiver, f.Name)
	case ti.NeedsNetip:
		return fmt.Sprintf("%s.%s.String()", receiver, f.Name)
	case ti.GoType == "bool":
		return fmt.Sprintf("fmt.Sprintf(\"%%t\", %s.%s)", receiver, f.Name)
	default: // numeric
		return fmt.Sprintf("fmt.Sprintf(\"%%d\", %s.%s)", receiver, f.Name)
	}
}

// formatLiteral renders a test-data value as a Go literal appropriate for the field type.
func formatLiteral(typeName string, v interface{}) string {
	ti := info(typeName)
	s := fmt.Sprintf("%v", v)
	switch {
	case ti.IsString:
		return fmt.Sprintf("%q", s)
	case ti.NeedsNetip:
		return fmt.Sprintf("netip.MustParseAddr(%q)", s)
	default:
		return s // bool / numeric literal
	}
}

// zeroLiteral renders the zero value of the field type as a Go literal.
func zeroLiteral(typeName string) string {
	ti := info(typeName)
	switch {
	case ti.IsString:
		return `""`
	case ti.NeedsNetip:
		return "netip.Addr{}"
	case ti.GoType == "bool":
		return "false"
	default:
		return "0"
	}
}

func main() {
	yamlFile, err := os.ReadFile("types_generate.yaml")
	if err != nil {
		log.Fatalf("Failed to read types_generate.yaml: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	for _, t := range config.Types {
		if err := generateTypeFile(&t); err != nil {
			log.Fatalf("Failed to generate type file for %s: %v", t.Name, err)
		}
		if err := generateTestFile(&t); err != nil {
			log.Fatalf("Failed to generate test file for %s: %v", t.Name, err)
		}
		if err := generateRdataFile(&t); err != nil {
			log.Fatalf("Failed to generate rdata file for %s: %v", t.Name, err)
		}
	}

	fmt.Println("Code generation complete!")
}

func toConstName(name string) string {
	s := strings.ToUpper(name)
	return strings.ReplaceAll(s, "_", "")
}

func toTypeName(name string) string {
	return toConstName(name)
}

func toFileName(name string) string {
	return strings.ToLower(name)
}

func camelCaseFromSnake(s string) string {
	parts := strings.Split(s, "_")
	var result []string
	for _, part := range parts {
		if len(part) > 0 {
			result = append(result, strings.ToUpper(part[:1])+strings.ToLower(part[1:]))
		}
	}
	return strings.Join(result, "")
}

func toDisplayName(name string) string {
	// if strings.HasPrefix(name, "Cf_Worker_Route") {
	// 	fmt.Printf("DEBUG: toDisplayName name=%q out=%q\n", name, strings.ToUpper(name))
	// }
	return strings.ToUpper(name)
}

func generateTypeFile(t *TypeDef) error {
	constName := toConstName(t.Name)
	typeName := toTypeName(t.Name)
	fileName := toFileName(t.Name)
	displayName := toDisplayName(t.Name)

	var buf bytes.Buffer

	buf.WriteString("package privatetypes\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"strconv\"\n")
	if anyNeedsNetip(t.Fields) {
		buf.WriteString("\t\"net/netip\"\n")
	}
	buf.WriteString("\n")
	buf.WriteString("\tdnsv2 \"codeberg.org/miekg/dns\"\n")
	buf.WriteString("\tdnsutilv2 \"codeberg.org/miekg/dns/dnsutil\"\n")

	if len(t.Fields) > 0 {
		buf.WriteString("\t\"github.com/DNSControl/dnscontrol/v4/pkg/mustbe\"\n")
	}

	buf.WriteString("\tprivatetypesrdata \"github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata\"\n")
	buf.WriteString(")\n\n")

	fmt.Fprintf(&buf, "// %s\n\n", displayName)

	fmt.Fprintf(&buf, "func init() {\n")
	fmt.Fprintf(&buf, "\tRegister(Type%s, \"%s\", func() dnsv2.RR { return new(%s) }, privatetypesrdata.Make%s)\n",
		constName, displayName, typeName, typeName)
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "const Type%s = uint16(%d)\n\n", constName, t.Codepoint)

	fmt.Fprintf(&buf, "type %s struct {\n", typeName)
	buf.WriteString("\tHdr dnsv2.Header\n\n")
	fmt.Fprintf(&buf, "\tprivatetypesrdata.%s\n", typeName)

	if len(t.Fields) > 0 {
		for _, f := range t.Fields {
			fmt.Fprintf(&buf, "\t// %-20s %s\n", f.Name, info(f.Type).GoType)
		}
	}

	buf.WriteString("}\n\n")

	buf.WriteString("// Typer interface.\n\n")
	fmt.Fprintf(&buf, "func (rr *%s) Type() uint16 { return Type%s }\n\n", typeName, constName)

	buf.WriteString("// RR interface.\n\n")
	fmt.Fprintf(&buf, "func (rr *%s) Header() *dnsv2.Header { return &rr.Hdr }\n", typeName)

	fmt.Fprintf(&buf, "func (rr *%s) Len() int {\n", typeName)
	if len(t.Fields) == 0 {
		buf.WriteString("\treturn rr.Hdr.Len()\n")
	} else {
		buf.WriteString("\treturn rr.Hdr.Len() + rr.Data().Len()\n")
	}
	buf.WriteString("}\n")

	fmt.Fprintf(&buf, "func (rr *%s) Data() dnsv2.RDATA {\n", typeName)
	if len(t.Fields) == 0 {
		fmt.Fprintf(&buf, "\treturn &privatetypesrdata.%s{}\n", typeName)
	} else {
		fmt.Fprintf(&buf, "\treturn &privatetypesrdata.%s{", typeName)
		for i, f := range t.Fields {
			if i > 0 {
				buf.WriteString(", ")
			}
			fmt.Fprintf(&buf, "%s: rr.%s", f.Name, f.Name)
		}
		buf.WriteString("}\n")
	}
	buf.WriteString("}\n")

	fmt.Fprintf(&buf, "func (rr *%s) Clone() dnsv2.RR {\n", typeName)
	if len(t.Fields) == 0 {
		fmt.Fprintf(&buf, "\treturn &%s{\n", typeName)
		buf.WriteString("\t\trr.Hdr,\n")
		fmt.Fprintf(&buf, "\t\tprivatetypesrdata.%s{}}\n", typeName)
	} else {
		fmt.Fprintf(&buf, "\treturn &%s{\n", typeName)
		buf.WriteString("\t\tHdr: rr.Hdr,\n")
		fmt.Fprintf(&buf, "\t\t%s: privatetypesrdata.%s{\n", typeName, typeName)
		for _, f := range t.Fields {
			fmt.Fprintf(&buf, "\t\t\t%s: rr.%s,\n", f.Name, f.Name)
		}
		buf.WriteString("\t\t}}\n")
	}
	buf.WriteString("}\n")

	fmt.Fprintf(&buf, "func (rr *%s) String() string {\n", typeName)
	if len(t.Fields) == 0 {
		fmt.Fprintf(&buf, "\treturn rr.Header().Name + \"\\t\" +\n")
		buf.WriteString("\t\tstrconv.FormatInt(int64(rr.Header().TTL), 10) + \"\\t\" +\n")
		fmt.Fprintf(&buf, "\t\tdnsutilv2.ClassToString(rr.Header().Class) + \"\\t%s\" // RDATA is empty.\n", displayName)
	} else {
		fmt.Fprintf(&buf, "\treturn (rr.Header().Name + \"\\t\" +\n")
		buf.WriteString("\t\tstrconv.FormatInt(int64(rr.Header().TTL), 10) + \"\\t\" +\n")
		fmt.Fprintf(&buf, "\t\tdnsutilv2.ClassToString(rr.Header().Class) + \"\\t%s\\t\" + rr.Data().String())\n", displayName)
	}
	buf.WriteString("}\n\n")

	fmt.Fprintf(&buf, "// Parse makes an RDATA for this type using the tokens from dnsv2's parser.\n")
	fmt.Fprintf(&buf, "func (rr *%s) Parse(tokens []string, s string) error {\n", typeName)
	buf.WriteString("\targs := TokensToArgs(tokens)\n")

	if len(t.Fields) == 0 {
		fmt.Fprintf(&buf, "\tif len(args) != 0 {\n")
		fmt.Fprintf(&buf, "\t\treturn fmt.Errorf(\"%s requires exactly 0 arguments, got %%d\", len(args))\n", displayName)
	} else {
		fmt.Fprintf(&buf, "\tif len(args) != %d {\n", len(t.Fields))
		fmt.Fprintf(&buf, "\t\treturn fmt.Errorf(\"%s requires exactly %d arguments, got %%d: %%v\", len(args), args)\n", displayName, len(t.Fields))
	}

	buf.WriteString("\t}\n")

	for i, f := range t.Fields {
		ti := info(f.Type)
		if ti.NeedsOrigin {
			fmt.Fprintf(&buf, "\trr.%s = mustbe.%s(\"\", args[%d])\n", f.Name, f.Type, i)
		} else {
			fmt.Fprintf(&buf, "\trr.%s = mustbe.%s(args[%d])\n", f.Name, f.Type, i)
		}
	}

	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n")

	return os.WriteFile(fmt.Sprintf("t_%s.go", fileName), buf.Bytes(), 0o644)
}

func generateTestFile(t *TypeDef) error {
	fileName := toFileName(t.Name)
	typeName := toTypeName(t.Name)
	displayName := toDisplayName(t.Name)
	testFuncName := camelCaseFromSnake(t.Name)

	var buf bytes.Buffer

	buf.WriteString("package privatetypes\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"testing\"\n")
	if anyNeedsNetip(t.Fields) {
		buf.WriteString("\t\"net/netip\"\n")
	}
	buf.WriteString("\n")
	buf.WriteString("\tdnsv2 \"codeberg.org/miekg/dns\"\n")

	if len(t.Fields) > 0 {
		buf.WriteString("\tprivatetypesrdata \"github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata\"\n")
	}

	buf.WriteString(")\n\n")

	if len(t.Fields) == 0 {
		fmt.Fprintf(&buf, "func Test%s(t *testing.T) {\n", testFuncName)
		fmt.Fprintf(&buf, "\ty := &%s{Hdr: dnsv2.Header{Name: \"example.org.\", Class: dnsv2.ClassINET}}\n", typeName)
		buf.WriteString("\trry, err := dnsv2.New(y.String())\n")
		buf.WriteString("\tif err != nil {\n")
		buf.WriteString("\t\tt.Fatal(err)\n")
		buf.WriteString("\t}\n")
		buf.WriteString("\tif rry.String() != y.String() {\n")
		fmt.Fprintf(&buf, "\t\tt.Fatalf(\"%s string presentations should be identical:\\n%%q\\n%%q\", rry.String(), y.String())\n", displayName)
		buf.WriteString("\t}\n")
		buf.WriteString("}\n")
	} else {
		if len(t.TestData) == 0 {
			fmt.Fprintf(&buf, "func Test%s(t *testing.T) {\n", testFuncName)
			fmt.Fprintf(&buf, "\ty := &%s{\n", typeName)
			buf.WriteString("\t\tHdr: dnsv2.Header{Name: \"example.org.\", Class: dnsv2.ClassINET},\n")
			fmt.Fprintf(&buf, "\t\t%s: privatetypesrdata.%s{\n", typeName, typeName)
			for _, f := range t.Fields {
				fmt.Fprintf(&buf, "\t\t\t%s: %s,\n", f.Name, zeroLiteral(f.Type))
			}
			buf.WriteString("\t\t},\n")
			buf.WriteString("\t}\n")
			buf.WriteString("\trry, err := dnsv2.New(y.String())\n")
			buf.WriteString("\tif err != nil {\n")
			buf.WriteString("\t\tt.Fatal(err)\n")
			buf.WriteString("\t}\n")
			buf.WriteString("\tif rry.String() != y.String() {\n")
			fmt.Fprintf(&buf, "\t\tt.Fatalf(\"%s string presentations should be identical:\\n%%s\\n%%s\", rry.String(), y.String())\n", displayName)
			buf.WriteString("\t}\n")
			buf.WriteString("}\n")
		} else {
			for _, td := range t.TestData {
				testName := testFuncName
				if td.Name != "" {
					testName = testFuncName + "_" + camelCaseFromSnake(td.Name)
				}

				fmt.Fprintf(&buf, "func Test%s(t *testing.T) {\n", testName)
				fmt.Fprintf(&buf, "\ty := &%s{\n", typeName)
				buf.WriteString("\t\tHdr: dnsv2.Header{Name: \"example.org.\", Class: dnsv2.ClassINET},\n")
				fmt.Fprintf(&buf, "\t\t%s: privatetypesrdata.%s{\n", typeName, typeName)

				for _, f := range t.Fields {
					var lit string
					if v, ok := td.Values[f.Name]; ok {
						lit = formatLiteral(f.Type, v)
					} else {
						lit = zeroLiteral(f.Type)
					}
					fmt.Fprintf(&buf, "\t\t\t%s: %s,\n", f.Name, lit)
				}

				buf.WriteString("\t\t},\n")
				buf.WriteString("\t}\n")
				buf.WriteString("\trry, err := dnsv2.New(y.String())\n")
				buf.WriteString("\tif err != nil {\n")
				buf.WriteString("\t\tt.Fatal(err)\n")
				buf.WriteString("\t}\n")
				buf.WriteString("\tif rry.String() != y.String() {\n")
				fmt.Fprintf(&buf, "\t\tt.Fatalf(\"%s string presentations should be identical:\\n%%s\\n%%s\", rry.String(), y.String())\n", displayName)
				buf.WriteString("\t}\n")
				buf.WriteString("}\n")
				if len(t.TestData) > 1 {
					buf.WriteString("\n")
				}
			}
		}
	}

	return os.WriteFile(fmt.Sprintf("t_%s_test.go", fileName), buf.Bytes(), 0o644)
}

func generateRdataFile(t *TypeDef) error {
	fileName := toFileName(t.Name)
	typeName := toTypeName(t.Name)
	displayName := toDisplayName(t.Name)

	var buf bytes.Buffer

	buf.WriteString("package privatetypesrdata\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"fmt\"\n")
	if anyNeedsNetip(t.Fields) {
		buf.WriteString("\t\"net/netip\"\n")
	}
	buf.WriteString("\n")
	buf.WriteString("\tdnsv2 \"codeberg.org/miekg/dns\"\n")
	buf.WriteString("\t\"github.com/DNSControl/dnscontrol/v4/pkg/mustbe\"\n")

	if len(t.Fields) > 0 {
		buf.WriteString("\t\"github.com/DNSControl/dnscontrol/v4/pkg/txtutil\"\n")
	}

	buf.WriteString(")\n\n")

	// Silence unused-import warnings when no field needs them.
	_ = anyNonString

	fmt.Fprintf(&buf, "type %s struct {\n", typeName)
	for _, f := range t.Fields {
		fmt.Fprintf(&buf, "\t%-20s %s\n", f.Name, info(f.Type).GoType)
	}
	buf.WriteString("}\n\n")

	// Len: bytes of the textual representation.
	fmt.Fprintf(&buf, "func (rd %s) Len() int {\n", typeName)
	if len(t.Fields) == 0 {
		buf.WriteString("\treturn 0\n")
	} else {
		buf.WriteString("\treturn len(rd.String())\n")
	}
	buf.WriteString("}\n\n")

	// String: Zoneify the field values.
	fmt.Fprintf(&buf, "func (rd %s) String() string {\n", typeName)
	if len(t.Fields) == 0 {
		buf.WriteString("\treturn \"\"\n")
	} else {
		buf.WriteString("\treturn txtutil.Zoneify([]string{")
		for i, f := range t.Fields {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fieldStringExpr("rd", f))
		}
		buf.WriteString("})\n")
	}
	buf.WriteString("}\n\n")

	// Make: validate arg count, then build the rdata using mustbe.X conversions.
	fmt.Fprintf(&buf, "func Make%s(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {\n", typeName)
	buf.WriteString("\tmustbe.ValidArgs(args)\n")

	if len(t.Fields) == 0 {
		fmt.Fprintf(&buf, "\tif len(args) != 0 {\n")
		fmt.Fprintf(&buf, "\t\treturn %s{}, fmt.Errorf(\"%s expects 0 arguments, got %%d: %%+v\", len(args), args)\n", typeName, displayName)
	} else {
		fmt.Fprintf(&buf, "\tif len(args) != %d {\n", len(t.Fields))
		fmt.Fprintf(&buf, "\t\treturn %s{}, fmt.Errorf(\"%s expects %d arguments, got %%d: %%+v\", len(args), args)\n", typeName, displayName, len(t.Fields))
	}

	buf.WriteString("\t}\n")

	if len(t.Fields) == 0 {
		fmt.Fprintf(&buf, "\treturn %s{}, nil\n", typeName)
	} else {
		fmt.Fprintf(&buf, "\treturn %s{\n", typeName)
		for i, f := range t.Fields {
			ti := info(f.Type)
			if ti.NeedsOrigin {
				fmt.Fprintf(&buf, "\t\t%s: mustbe.%s(origin, args[%d]),\n", f.Name, f.Type, i)
			} else {
				fmt.Fprintf(&buf, "\t\t%s: mustbe.%s(args[%d]),\n", f.Name, f.Type, i)
			}
		}
		buf.WriteString("\t}, nil\n")
	}

	buf.WriteString("}\n")

	// "origin" is unused when there are no TargetHost fields.
	if len(t.Fields) > 0 {
		needsOrigin := false
		for _, f := range t.Fields {
			if info(f.Type).NeedsOrigin {
				needsOrigin = true
				break
			}
		}
		if !needsOrigin {
			// Rewrite the receiver to use _ instead of origin to avoid unused-var warnings.
			out := bytes.Replace(buf.Bytes(),
				[]byte(fmt.Sprintf("func Make%s(origin string, args ...any)", typeName)),
				[]byte(fmt.Sprintf("func Make%s(_ string, args ...any)", typeName)),
				1)
			buf.Reset()
			buf.Write(out)
		}
	}

	os.MkdirAll("rdata", 0o755)

	return os.WriteFile(filepath.Join("rdata", fmt.Sprintf("rdata_%s.go", fileName)), buf.Bytes(), 0o644)
}

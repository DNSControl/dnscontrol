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

func main() {
	// Read the YAML file
	yamlFile, err := os.ReadFile("types_generate.yaml")
	if err != nil {
		log.Fatalf("Failed to read types_generate.yaml: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	// Generate files for each type
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
	// Convert Adguardhome_A_Passthrough to ADGUARDHOMEAPASSTHROUGH
	s := strings.ToUpper(name)
	return strings.ReplaceAll(s, "_", "")
}

func toTypeName(name string) string {
	// Convert Adguardhome_A_Passthrough to ADGUARDHOMEAPASSTHROUGH (same as const name for simplicity)
	return toConstName(name)
}

func toFileName(name string) string {
	// Convert Adguardhome_A_Passthrough to adguardhome_a_passthrough
	return strings.ToLower(name)
}

func camelCaseFromSnake(s string) string {
	// Convert Adguardhome_A_Passthrough to AdguardhomeAPassthrough
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
	// Convert Adguardhome_A_Passthrough to ADGUARDHOME_A_PASSTHROUGH
	return strings.ToUpper(name)
}

func generateTypeFile(t *TypeDef) error {
	constName := toConstName(t.Name)
	typeName := toTypeName(t.Name)
	fileName := toFileName(t.Name)
	displayName := toDisplayName(t.Name)

	var buf bytes.Buffer

	// Package and imports
	buf.WriteString("package privatetypes\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"strconv\"\n")
	buf.WriteString("\n")
	buf.WriteString("\tdnsv2 \"codeberg.org/miekg/dns\"\n")
	buf.WriteString("\tdnsutilv2 \"codeberg.org/miekg/dns/dnsutil\"\n")

	if len(t.Fields) > 0 {
		buf.WriteString("\t\"github.com/DNSControl/dnscontrol/v4/pkg/mustbe\"\n")
	}

	buf.WriteString("\tprivatetypesrdata \"github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata\"\n")
	buf.WriteString(")\n\n")

	// Comment
	fmt.Fprintf(&buf, "// %s\n\n", displayName)

	// init function
	fmt.Fprintf(&buf, "func init() {\n")
	fmt.Fprintf(&buf, "\tRegister(Type%s, \"%s\", func() dnsv2.RR { return new(%s) }, privatetypesrdata.Make%s)\n", constName, displayName, typeName, typeName)
	fmt.Fprintf(&buf, "}\n\n")

	// Constant
	fmt.Fprintf(&buf, "const Type%s = %d\n\n", constName, t.Codepoint)

	// Type definition
	fmt.Fprintf(&buf, "type %s struct {\n", typeName)
	buf.WriteString("\tHdr dnsv2.Header\n\n")
	fmt.Fprintf(&buf, "\tprivatetypesrdata.%s\n", typeName)

	// Comment for fields (optional)
	if len(t.Fields) > 0 {
		for _, f := range t.Fields {
			fmt.Fprintf(&buf, "\t// %-20s string\n", f.Name)
		}
	}

	buf.WriteString("}\n\n")

	// Typer interface
	buf.WriteString("// Typer interface.\n\n")
	fmt.Fprintf(&buf, "func (rr *%s) Type() uint16 { return Type%s }\n\n", typeName, constName)

	// RR interface
	buf.WriteString("// RR interface.\n\n")
	fmt.Fprintf(&buf, "func (rr *%s) Header() *dnsv2.Header { return &rr.Hdr }\n", typeName)

	// Len method
	fmt.Fprintf(&buf, "func (rr *%s) Len() int {\n", typeName)
	if len(t.Fields) == 0 {
		buf.WriteString("\treturn rr.Hdr.Len()\n")
	} else {
		buf.WriteString("\treturn rr.Hdr.Len() +\n")
		for i, f := range t.Fields {
			fmt.Fprintf(&buf, "\t\t1 + len(rr.%s)", f.Name)
			if i < len(t.Fields)-1 {
				buf.WriteString(" +")
			}
			buf.WriteString("\n")
		}
	}
	buf.WriteString("}\n")

	// Data method
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

	// Clone method
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
			fmt.Fprintf(&buf, "\t\t\t%s:        rr.%s,\n", f.Name, f.Name)
		}
		buf.WriteString("\t\t}}\n")
	}
	buf.WriteString("}\n")

	// String method
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

	// Parse method
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

	// Parse field assignments
	for i, f := range t.Fields {
		if f.Type == "RawString" {
			fmt.Fprintf(&buf, "\trr.%s = mustbe.%s(args[%d])\n", f.Name, f.Type, i)
		} else {
			fmt.Fprintf(&buf, "\trr.%s = mustbe.%s(\"\", args[%d])\n", f.Name, f.Type, i)
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
	buf.WriteString("\t\"testing\"\n\n")
	buf.WriteString("\tdnsv2 \"codeberg.org/miekg/dns\"\n")

	if len(t.Fields) > 0 {
		buf.WriteString("\tprivatetypesrdata \"github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata\"\n")
	}

	buf.WriteString(")\n\n")

	// For types with no fields, generate a single test
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
		// For types with fields, generate one test per test_data entry
		if len(t.TestData) == 0 {
			// If no test data, generate one test with empty strings
			fmt.Fprintf(&buf, "func Test%s(t *testing.T) {\n", testFuncName)
			fmt.Fprintf(&buf, "\ty := &%s{\n", typeName)
			buf.WriteString("\t\tHdr: dnsv2.Header{Name: \"example.org.\", Class: dnsv2.ClassINET},\n")
			fmt.Fprintf(&buf, "\t\t%s: privatetypesrdata.%s{\n", typeName, typeName)
			for _, f := range t.Fields {
				fmt.Fprintf(&buf, "\t\t\t%s:        \"\",\n", f.Name)
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
			// Generate one test function per test data entry
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
					val := ""
					if v, ok := td.Values[f.Name]; ok {
						val = fmt.Sprintf("%v", v)
					}
					fmt.Fprintf(&buf, "\t\t\t%s:        %q,\n", f.Name, val)
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
	buf.WriteString("\t\"fmt\"\n\n")
	buf.WriteString("\tdnsv2 \"codeberg.org/miekg/dns\"\n")

	if len(t.Fields) > 0 {
		buf.WriteString("\t\"github.com/DNSControl/dnscontrol/v4/pkg/mustbe\"\n")
		buf.WriteString("\t\"github.com/DNSControl/dnscontrol/v4/pkg/txtutil\"\n")
	}

	buf.WriteString(")\n\n")

	// Type definition
	fmt.Fprintf(&buf, "type %s struct {\n", typeName)

	for _, f := range t.Fields {
		fmt.Fprintf(&buf, "\t%-20s string\n", f.Name)
	}

	buf.WriteString("}\n\n")

	// Len method
	fmt.Fprintf(&buf, "func (rd %s) Len() int {\n", typeName)

	if len(t.Fields) == 0 {
		buf.WriteString("\treturn 0\n")
	} else {
		for i, f := range t.Fields {
			if i == 0 {
				fmt.Fprintf(&buf, "\treturn len(rd.%s)", f.Name)
			} else {
				fmt.Fprintf(&buf, " +\n\t\t1 + len(rd.%s)", f.Name)
			}
		}
		buf.WriteString("\n")
	}
	buf.WriteString("}\n\n")

	// String method
	fmt.Fprintf(&buf, "func (rd %s) String() string {\n", typeName)

	if len(t.Fields) == 0 {
		buf.WriteString("\treturn \"\"\n")
	} else {
		buf.WriteString("\treturn txtutil.Zoneify([]string{")
		for i, f := range t.Fields {
			if i > 0 {
				buf.WriteString(", ")
			}
			fmt.Fprintf(&buf, "rd.%s", f.Name)
		}
		buf.WriteString("})\n")
	}

	buf.WriteString("}\n\n")

	// Make function
	fmt.Fprintf(&buf, "func Make%s(origin string, args ...any) (dnsv2.RDATA, error) {\n", typeName)

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
		fmt.Fprintf(&buf, "\treturn %s{", typeName)
		for i, f := range t.Fields {
			if i > 0 {
				buf.WriteString(", ")
			}
			if f.Type == "RawString" {
				fmt.Fprintf(&buf, "mustbe.%s(args[%d])", f.Type, i)
			} else {
				fmt.Fprintf(&buf, "mustbe.%s(\"\", args[%d])", f.Type, i)
			}
		}
		buf.WriteString("}, nil\n")
	}

	buf.WriteString("}\n")

	// Ensure rdata directory exists
	os.MkdirAll("rdata", 0o755)

	return os.WriteFile(filepath.Join("rdata", fmt.Sprintf("rdata_%s.go", fileName)), buf.Bytes(), 0o644)
}

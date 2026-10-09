package commands

import (
	"os"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func TestInitDocumentationLinks(t *testing.T) {
	for _, vendorURL := range []string{"", "https://api.example.test/docs"} {
		t.Run("vendor="+vendorURL, func(t *testing.T) {
			output := captureInitOutput(t)
			registered, _ := providers.GetDefinition(providerSyntaxTestType)
			def := *registered
			def.VendorAPIDocURL = vendorURL
			if _, _, err := askEntry(&stubAsker{t: t}, &def, "account"); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			got := string(data)
			if !strings.Contains(got, "DNSControl documentation: https://docs.dnscontrol.org/provider/test_provider_syntax\n") {
				t.Fatal("init did not display the derived documentation URL")
			}
			if vendorURL == "" {
				if strings.Contains(got, "Vendor API documentation:") {
					t.Fatal("init displayed an empty vendor documentation link")
				}
			} else if !strings.Contains(got, "Vendor API documentation: "+vendorURL+"\n") {
				t.Fatal("init did not display the vendor API documentation link")
			}
		})
	}
}

func TestFieldLabel(t *testing.T) {
	tests := []struct {
		name  string
		field providers.CredsField
		want  string
	}{
		{
			name:  "required field with label",
			field: providers.CredsField{Key: "apitoken", Label: "API Token", Required: true},
			want:  "API Token [apitoken] (required)",
		},
		{
			name:  "optional field without label",
			field: providers.CredsField{Key: "apitoken"},
			want:  "apitoken (optional)",
		},
		{
			name:  "label equal to key",
			field: providers.CredsField{Key: "username", Label: "Username"},
			want:  "Username (optional)",
		},
		{
			name:  "optional suffix in label",
			field: providers.CredsField{Key: "sandbox", Label: "Use sandbox (optional)"},
			want:  "Use sandbox [sandbox] (optional)",
		},
		{
			name:  "required suffix in label",
			field: providers.CredsField{Key: "apikey", Label: "API key (required)", Required: true},
			want:  "API key [apikey] (required)",
		},
		{
			name:  "yes/no field without suffix",
			field: providers.CredsField{Key: "sandbox", Label: "Use the sandbox API?", ConfirmValue: "1"},
			want:  "Use the sandbox API? [sandbox]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fieldLabel(tt.field); got != tt.want {
				t.Errorf("fieldLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAskFieldConfirmValue(t *testing.T) {
	tests := []struct {
		name    string
		confirm bool
		want    string
	}{
		{name: "yes stores the confirm value", confirm: true, want: "1"},
		{name: "no leaves the field empty", confirm: false, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asker := &stubAsker{t: t, confirm: []bool{tt.confirm}}
			field := providers.CredsField{Key: "sandbox", Label: "Use the sandbox API?", ConfirmValue: "1"}
			got, err := askField(asker, field)
			if err != nil {
				t.Fatalf("askField() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("askField() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCollectFieldsConfirmValueNo(t *testing.T) {
	asker := &stubAsker{t: t, confirm: []bool{false}}
	meta := &providers.Definition{
		CredFields: []providers.CredsField{
			{Key: "sandbox", Label: "Use the sandbox API?", ConfirmValue: "1"},
		},
	}
	got, err := collectFields(asker, meta)
	if err != nil {
		t.Fatalf("collectFields() error = %v", err)
	}
	if _, found := got["sandbox"]; found {
		t.Errorf("collectFields() wrote sandbox = %q, want the key left out", got["sandbox"])
	}
}

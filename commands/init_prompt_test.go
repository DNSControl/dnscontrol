package commands

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fieldLabel(tt.field); got != tt.want {
				t.Errorf("fieldLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

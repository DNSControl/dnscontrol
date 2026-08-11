package cloudflare

import (
	"testing"
)

func TestParseCfTxtContent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "unquoted string",
			input:   "simple text",
			want:    "simple text",
			wantErr: false,
		},
		{
			name:    "quoted string",
			input:   `"quoted text"`,
			want:    "quoted text",
			wantErr: false,
		},
		{
			name:    "quoted string with spaces",
			input:   `"text with spaces"`,
			want:    "text with spaces",
			wantErr: false,
		},
		{
			name:    "only opening quote",
			input:   `"incomplete`,
			want:    `"incomplete`,
			wantErr: false,
		},
		{
			name:    "only closing quote",
			input:   `incomplete"`,
			want:    `incomplete"`,
			wantErr: false,
		},
		{
			name:    "single quote char",
			input:   `"`,
			want:    ``,
			wantErr: true,
		},
		{
			name:    "double quotes only",
			input:   `""`,
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCfTxtContent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCfTxtContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseCfTxtContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

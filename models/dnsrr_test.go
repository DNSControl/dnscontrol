package models

import "testing"

func TestStringWithMeta(t *testing.T) {
	dc := MustNewDomainConfig("example.com")

	tests := []struct {
		name     string
		rc       *RecordConfig
		metadata map[string]string
		want     string
	}{
		{
			name: "A",
			rc:   dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1"),
			want: "www 300 IN A 192.0.2.1",
		},
		{
			name: "zero TTL is included",
			rc:   dc.MustNewRecordConfig("www", 0, "A", "192.0.2.1"),
			want: "www 0 IN A 192.0.2.1",
		},
		{
			name: "MX at the apex",
			rc:   dc.MustNewRecordConfig("@", 3600, "MX", 10, "mail.example.com."),
			want: "@ 3600 IN MX 10 mail.example.com.",
		},
		{
			name:     "metadata is sorted and quoted",
			rc:       dc.MustNewRecordConfig("fwd", 300, "A", "192.0.2.1"),
			metadata: map[string]string{"wildcard": "no", "includePath": "yes"},
			want:     `fwd 300 IN A 192.0.2.1 ; includePath="yes" wildcard="no"`,
		},
		{
			name:     "metadata value with a space and a quote",
			rc:       dc.MustNewRecordConfig("fwd", 300, "A", "192.0.2.1"),
			metadata: map[string]string{"note": `a "b" c`},
			want:     `fwd 300 IN A 192.0.2.1 ; note="a \"b\" c"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metadata != nil {
				tt.rc.Metadata = tt.metadata
			}
			if got := tt.rc.StringWithMeta(); got != tt.want {
				t.Errorf("StringWithMeta() = %q, want %q", got, tt.want)
			}
		})
	}
}

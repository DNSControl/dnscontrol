package mustbe_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/mustbe"
)

func TestOpenPGPKey(t *testing.T) {
	tests := []struct {
		name    string
		a       any
		want    string
		wantErr bool
	}{
		{"base64", "mQENBA==", "mQENBA==", false},
		{"hex lower", "99010d04", "mQENBA==", false},
		{"hex upper", "99010D04", "mQENBA==", false},
		{"neither", "not a key!", "", true},
		{"base64 with whitespace", "mQEN BA==", "", true},
		{"wrong type", 42.0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mustbe.OpenPGPKey(tt.a)
			if (err != nil) != tt.wantErr {
				t.Fatalf("OpenPGPKey(%v) err = %v, wantErr %v", tt.a, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("OpenPGPKey(%v) = %q, want %q", tt.a, got, tt.want)
			}
			if tt.wantErr {
				return
			}
			// Round Trip
			second, err := mustbe.OpenPGPKey(got)
			if err != nil || second != got {
				t.Errorf("OpenPGPKey(%q) round trip = %q, %v", got, second, err)
			}
		})
	}
}

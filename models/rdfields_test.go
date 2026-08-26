package models_test

import (
	"slices"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestRDtoFieldsJS(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		s       dnsv2.RDATA
		want    []string
		wantErr bool
	}{
		{"mx", dnsrdatav2.MX{Preference: 10, Mx: "foo.example.com."}, []string{"10", `"foo.example.com."`}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := models.RDtoFieldsJS(tt.s)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("RDtoFieldsJS() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("RDtoFieldsJS() succeeded unexpectedly")
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("RDtoFieldsJS() = %v, want %v", got, tt.want)
			}
		})
	}
}

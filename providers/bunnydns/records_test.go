package bunnydns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestComparableFuncSmartRoutingMetadata(t *testing.T) {
	tests := []struct {
		name     string
		recType  string
		metadata map[string]string
		want     string
	}{
		{
			name:    "geographic A ignores unrelated fields",
			recType: "A",
			metadata: map[string]string{
				metaSmartRoutingType:     "geographic",
				metaGeolocationLatitude:  "40.7128",
				metaGeolocationLongitude: "-74.006",
				"unrelated":              "ignored",
			},
			want: `{"bunny_geolocation_latitude":"40.7128","bunny_geolocation_longitude":"-74.006","bunny_smart_routing_type":"geographic"}`,
		},
		{
			name:    "latency AAAA",
			recType: "AAAA",
			metadata: map[string]string{
				metaSmartRoutingType: "latency",
				metaLatencyZone:      "NY",
			},
			want: `{"bunny_latency_zone":"NY","bunny_smart_routing_type":"latency"}`,
		},
		{
			name:     "unrelated metadata only",
			recType:  "A",
			metadata: map[string]string{"unrelated": "ignored"},
		},
		{
			name:    "unsupported record type",
			recType: "CNAME",
			metadata: map[string]string{
				metaSmartRoutingType: "latency",
				metaLatencyZone:      "NY",
			},
		},
		{
			name:    "no metadata",
			recType: "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := comparableFunc(&models.RecordConfig{Type: tt.recType, Metadata: tt.metadata})
			if got != tt.want {
				t.Errorf("comparableFunc() = %q, want %q", got, tt.want)
			}
		})
	}
}

package ovh

import (
	"testing"

	"github.com/ovh/go-ovh/ovh"
)

func Test_getOVHEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{
			"default to EU", "", ovh.OvhEU,
		},
		{
			"default to EU if omitted", "omitted", ovh.OvhEU,
		},
		{
			"set to EU", "eu", ovh.OvhEU,
		},
		{
			"set to CA", "ca", ovh.OvhCA,
		},
		{
			"set to US", "us", ovh.OvhUS,
		},
		{
			"case insensitive", "Eu", ovh.OvhEU,
		},
		{
			"case insensitive ca", "CA", ovh.OvhCA,
		},
		{
			"arbitratry", "https://blah", "https://blah",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := make(map[string]string)
			if tt.endpoint != "" && tt.endpoint != "omitted" {
				params["endpoint"] = tt.endpoint
			}
			if got := getOVHEndpoint(params); got != tt.want {
				t.Errorf("getOVHEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

package js

import (
	"fmt"
	"testing"
)

// TestDomainNameValidation verifies that invalid domain names and tags
// are rejected as soon as dnsconfig.js is processed. (GHSA-rrxr-r9rr-crr4)
func TestDomainNameValidation(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"example.com", false},
		{"example.com!inside", false},
		{"example.com!in_side-2", false},
		{"example.com!", false},
		{"0/26.2.0.192.in-addr.arpa", false},
		{"рф.com", false},
		{"bücher.example", false},
		{"例え.jp", false},
		{"münchen.de!inside", false},
		{"xn--p1ai.com", false},
		{"example.com.", true},
		{"../../outside/deep/owned", true},
		{"..", true},
		{".example.com", true},
		{"example..com", true},
		{"example.com!../../owned", true},
		{"example.com!a/b", true},
		{"example.com!v1.2", false},
		{"example.com!a..b", false},
		{"example.com!.", true},
		{"example.com!..", true},
		{"example.com!.hidden", true},
		{"example.com!a!b", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := fmt.Sprintf(`D(%q, NewRegistrar("none"), A("@", "1.2.3.4"));`, tt.name)
			_, err := ExecuteJavascriptString([]byte(script), true, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("D(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

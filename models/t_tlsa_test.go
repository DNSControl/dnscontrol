package models_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

func TestMakeTLSASMIMEA(t *testing.T) {
	makers := map[string]func(string, map[string]string, nrc.Flags, ...any) (any, error){
		"TLSA": func(o string, m map[string]string, f nrc.Flags, a ...any) (any, error) {
			return models.MakeTLSA(o, m, f, a...)
		},
		"SMIMEA": func(o string, m map[string]string, f nrc.Flags, a ...any) (any, error) {
			return models.MakeSMIMEA(o, m, f, a...)
		},
	}
	tests := []struct {
		name         string
		usage        int
		selector     int
		matchingType int
		isError      bool
	}{
		{"valid", 3, 1, 1, false},
		{"max values", 3, 1, 2, false},
		{"private use", 255, 255, 255, false},
		{"bad usage", 4, 1, 1, true},
		{"bad selector", 3, 2, 1, true},
		{"bad matching type", 3, 1, 3, true},
		{"254 usage", 254, 1, 1, true},
	}
	for rtype, maker := range makers {
		for _, tt := range tests {
			t.Run(rtype+" "+tt.name, func(t *testing.T) {
				// User input (dnsconfig.js) is checked.
				_, err := maker("example.com", nil, nrc.Flags{StrictParamChecking: true}, tt.usage, tt.selector, tt.matchingType, "abcdef01")
				if (err != nil) != tt.isError {
					t.Errorf("got err=%v, wantErr=%v", err, tt.isError)
				}
				// Provider data is accepted as-is.
				if _, err := maker("example.com", nil, nrc.Flags{}, tt.usage, tt.selector, tt.matchingType, "abcdef01"); err != nil {
					t.Errorf("unchecked: got err=%v, want nil", err)
				}
			})
		}
	}
}

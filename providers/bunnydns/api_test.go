package bunnydns

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestModifyRecordSmartRoutingType(t *testing.T) {
	originalClient := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = originalClient })

	tests := []struct {
		name   string
		typeID recordType
		route  smartRoutingType
		want   string
	}{
		{"disable A", recordTypeA, smartRoutingNone, "0"},
		{"disable AAAA", recordTypeAAAA, smartRoutingNone, "0"},
		{"latency A", recordTypeA, smartRoutingLatency, "1"},
		{"other record type", recordTypeCNAME, smartRoutingNone, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			http.DefaultClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				var err error
				body, err = io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
			})}
			if err := (&bunnydnsProvider{}).modifyRecord(1, 2, &record{Type: tt.typeID, SmartRoutingType: tt.route}); err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(body, &fields); err != nil {
				t.Fatal(err)
			}
			if got := string(fields["SmartRoutingType"]); got != tt.want {
				t.Errorf("SmartRoutingType in %s = %q, want %q", body, got, tt.want)
			}
		})
	}
}

package models

import "testing"

func Test_naptrQuoteFlag(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		contents string
		want     string
		wantErr  bool
	}{
		{"a", `100 10 U E2U+sip "!^.*$!sip:customer-service@example.com!" .`, `100 10 "U" "E2U+sip" "!^.*$!sip:customer-service@example.com!" .`, false},
		{"b", `100 10 U "E2U+sip" "!^.*$!sip:customer-service@example.com!" .`, `100 10 "U" "E2U+sip" "!^.*$!sip:customer-service@example.com!" .`, false},
		{"c", `100 10 "U" E2U+sip "!^.*$!sip:customer-service@example.com!" .`, `100 10 "U" "E2U+sip" "!^.*$!sip:customer-service@example.com!" .`, false},
		{"d", `100 10 "U" "E2U+sip" "!^.*$!sip:customer-service@example.com!" .`, `100 10 "U" "E2U+sip" "!^.*$!sip:customer-service@example.com!" .`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := naptrAssureQuotedFields(tt.contents)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("naptrQuoteFlag() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("naptrQuoteFlag() succeeded unexpectedly")
			}
			if got != tt.want {
				t.Errorf("naptrQuoteFlag(%q) = %v, want %v", tt.contents, got, tt.want)
			}
		})
	}
}

func Test_naptrFixFlag(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		flag    string
		want    string
		wantErr bool
	}{
		{"a", ``, ``, true},
		{"b", `U`, `"U"`, false},
		{"c", `"U"`, `"U"`, false},
		{"d", `AA`, `"AA"`, false},
		{"e", `"AA"`, `"AA"`, false},
		//
		{"ea", `"`, ``, true},
		{"eb", `"a`, ``, true},
		{"ec", `b"`, ``, true},
		{"ed", `"AA`, ``, true},
		{"ef", `AA"`, ``, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := naptrAddQuotes(tt.flag)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("naptrFixFlag() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("naptrFixFlag() succeeded unexpectedly")
			}
			if got != tt.want {
				t.Errorf("naptrFixFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

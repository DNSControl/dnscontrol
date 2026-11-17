package normalize

import (
	"fmt"
	"testing"
)

func TestCheckLabel(t *testing.T) {
	tests := []struct {
		label   string
		rType   string
		target  string
		isError bool
	}{
		{"@", "A", "zap", false},
		{"foo.bar", "A", "zap", false},
		{"_foo", "A", "zap", false},
		{"_foo", "SRV", "zap", false},
		{"_foo", "TLSA", "zap", false},
		{"_foo", "TXT", "zap", false},
		{"_y2", "CNAME", "foo", false},
		{"s1._domainkey", "CNAME", "foo", false},
		{"_y3", "CNAME", "asfljds.acm-validations.aws.", false},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.label, test.rType), func(t *testing.T) {
			err := checkLabel(test.label, test.rType, "foo.tld")
			if err != nil && !test.isError {
				t.Errorf("%02d: Expected no error but got %s", i, err)
			}
			if err == nil && test.isError {
				t.Errorf("%02d: Expected error but got none", i)
			}
		})
	}
}

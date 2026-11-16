package normalize

import (
	"fmt"
	"testing"
)

func TestCheckLabel(t *testing.T) {
	tests := []struct {
		label       string
		rType       string
		target      string
		isError     bool
		hasSkipMeta bool
	}{
		{"@", "A", "zap", false, false},
		{"foo.bar", "A", "zap", false, false},
		{"_foo", "A", "zap", false, false},
		{"_foo", "SRV", "zap", false, false},
		{"_foo", "TLSA", "zap", false, false},
		{"_foo", "TXT", "zap", false, false},
		{"_y2", "CNAME", "foo", false, false},
		{"s1._domainkey", "CNAME", "foo", false, false},
		{"_y3", "CNAME", "asfljds.acm-validations.aws.", false, false},
		{"test.foo.tld", "A", "zap", true, false},
		{"test.foo.tld", "A", "zap", false, true},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.label, test.rType), func(t *testing.T) {
			meta := map[string]string{}
			if test.hasSkipMeta {
				meta["skip_fqdn_check"] = "true"
			}
			err := checkLabel(test.label, test.rType, "foo.tld", meta)
			if err != nil && !test.isError {
				t.Errorf("%02d: Expected no error but got %s", i, err)
			}
			if err == nil && test.isError {
				t.Errorf("%02d: Expected error but got none", i)
			}
		})
	}
}

func Test_errorRepeat(t *testing.T) {
	type args struct {
		label  string
		domain string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "1",
			args: args{label: "foo.bar.com", domain: "bar.com"},
			want: `The name "foo.bar.com.bar.com." is an error (repeats the domain).` +
				` Maybe instead of "foo.bar.com" you intended "foo"?` +
				` If not add DISABLE_REPEATED_DOMAIN_CHECK to this record to permit this as-is.`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errorRepeat(tt.args.label, tt.args.domain); got != tt.want {
				t.Errorf("errorRepeat() = \n'%s', want\n'%s'", got, tt.want)
			}
		})
	}
}

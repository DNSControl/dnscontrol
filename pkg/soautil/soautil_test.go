package soautil

import (
	"reflect"
	"testing"
)

func Test_RFC5322MailToBind(t *testing.T) {
	tests := []struct {
		name        string
		rfc5322Mail string
		bindMail    string
	}{
		{"0", "hostmaster@example.com", "hostmaster.example.com"},
		{"1", "admin.dns@example.com", "admin\\.dns.example.com"},
		{"2", "hostmaster@sub.example.com", "hostmaster.sub.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RFC5322MailToBind(tt.rfc5322Mail); !reflect.DeepEqual(got, tt.bindMail) {
				t.Errorf("RFC5322MailToBind(%v) = %v, want %v", tt.rfc5322Mail, got, tt.bindMail)
			}
		})
	}
}

func Test_BindMailToRFC5322(t *testing.T) {
	tests := []struct {
		name        string
		bindMail    string
		rfc5322Mail string
	}{
		{"0", "hostmaster.example.com", "hostmaster@example.com"},
		{"1", "hostmaster.example.com.", "hostmaster@example.com"},
		{"2", "admin\\.dns.example.com", "admin.dns@example.com"},
		{"3", "hostmaster.sub.example.com", "hostmaster@sub.example.com"},
		{"4", "hostmaster", ""},
		{"5", "", ""},
		{"6", ".", ""},
		{"7", ".example.com", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BindMailToRFC5322(tt.bindMail); !reflect.DeepEqual(got, tt.rfc5322Mail) {
				t.Errorf("BindMailToRFC5322(%v) = %v, want %v", tt.bindMail, got, tt.rfc5322Mail)
			}
		})
	}
}

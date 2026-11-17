package mustbe

import (
	"errors"
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

// RRType upcases a field that names a DNS record type (such as the type a LUA
// record emits). origin is unused; it is accepted so that the generated code
// can call this like TargetHost.
// With isEnabled.StrictParamChecking, the result must be a known DNS type.
func RRType(_ string, isEnabled nrc.Flags, arg any) (string, error) {
	s := ToUpperRawString(arg)
	if isEnabled.StrictParamChecking {
		if s == "" {
			return "", errors.New("a record type must be specified")
		}
		if _, ok := dnsv2.StringToType[s]; !ok {
			return "", fmt.Errorf("record type (%s) is not a valid DNS type", s)
		}
	}
	return s, nil
}

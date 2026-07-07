package models

// methods that make RecordConfig meet the dns.RR interface.

import "github.com/DNSControl/dnscontrol/v4/pkg/txtutil"

// String returns the text representation of the resource record.
func (rc *RecordConfig) String() string {
	// TXT presentation splits the value into quoted, 255-octet character-strings.
	// GetTargetCombined() returns the raw text, so quote/chunk it here.
	if rc.Type == "TXT" {
		return txtutil.EncodeQuoted(rc.GetTargetTXTJoined())
	}
	return rc.GetTargetCombined()
}

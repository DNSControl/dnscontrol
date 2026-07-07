package models

import (
	"strings"
	"unicode"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

/*
Sadly many providers handle TXT records in strange and unexpeected
ways.  DNSControl has to handle all of them.  Over the years we've
tried many things.  This explain the current state of the code.

DNSControl stores the TXT record target as a single string of any length.
Providers take care of any splitting, excaping, or quoting.

NOTE: Older versions of DNSControl stored the TXT record as
represented by the provider, which could be a single string, a series
of smaller strings, or a single string that is quoted/escaped.  This
created tons of edge-cases and other distractions.

If a provider doesn't support certain characters in a TXT record, use
the providers/$PROVIDER/auditrecords.go file to indicate this.
DNSControl uses this information to warn users of unsupporrted input,
and to skip related integration tests.

There are 2 ways to create a TXT record:
	SetTargetTXT():  Create from a string.
	SetTargetTXTs(): Create from an array of strings that need to be joined.

There are 2 ways to get the value (target) of a TXT record:
	GetTargetTXTJoined(): Returns one big string
	GetTargetTXTSegmented(): Returns an array 255-octet segments.

*/

// HasFormatIdenticalToTXT returns if a RecordConfig has a format which is
// identical to TXT, such as SPF. For more details, read
// https://tools.ietf.org/html/rfc4408#section-3.1.1
func (rc *RecordConfig) HasFormatIdenticalToTXT() bool {
	return rc.Type == "TXT" || rc.Type == "SPF" || rc.Type == "LUA"
}

// SetTargetTXT sets the TXT fields when there is 1 string.
//
// LUA records reuse the TXT accessors (their payload is a TXT-format string).
// When .Type is "LUA" the value is stored as a LUA rdata (using the already-set
// .LuaRType) rather than a TXT rdata.
func (rc *RecordConfig) SetTargetTXT(s string) error {
	if rc.Type == "LUA" {
		rc.SetRDATA(privatetypesrdata.LUA{LuaType: rc.LuaRType, LuaPayload: s})
		return rc.SetTarget(s)
	}
	return legacySetTargetArgs(rc, dnsv2.TypeTXT, s)
}

// SetTargetTXTs sets the TXT fields when there are many strings. They are stored concatenated.
func (rc *RecordConfig) SetTargetTXTs(s []string) error {
	return rc.SetTargetTXT(strings.Join(s, ""))
}

// txtPayload returns the record's TXT payload as one string. It supports both
// TXT rdata and LUA rdata (whose payload is TXT-format).
func (rc *RecordConfig) txtPayload() string {
	switch rd := rc.GetRDATA().(type) {
	case dnsrdatav2.TXT:
		return strings.Join(rd.Txt, "")
	case privatetypesrdata.LUA:
		return rd.LuaPayload
	}
	return ""
}

// GetTargetTXTJoined returns the TXT target as one string.
func (rc *RecordConfig) GetTargetTXTJoined() string {
	return rc.txtPayload()
}

// GetTargetTXTSegmented returns the TXT target as 255-octet segments, with the remainder in the last segment.
func (rc *RecordConfig) GetTargetTXTSegmented() []string {
	return splitChunks(rc.txtPayload(), 255)
}

// GetTargetTXTSegmentCount returns the number of 255-octet segments required to store TXT target.
func (rc *RecordConfig) GetTargetTXTSegmentCount() int {
	total := len(rc.txtPayload())
	segs := total / 255 // integer division, decimals are truncated
	if (total % 255) > 0 {
		return segs + 1
	}
	return segs
}

func splitChunks(buf string, lim int) []string {
	if len(buf) == 0 {
		return nil
	}

	var chunk string
	chunks := make([]string, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf)
	}
	return chunks
}

// ParseLuaContent splits a PowerDNS LUA record content string into its emitted rtype and payload.
func ParseLuaContent(content string) (rtype string, payload string) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", ""
	}
	splitIndex := -1
	for i, r := range trimmed {
		if unicode.IsSpace(r) {
			splitIndex = i
			break
		}
	}
	if splitIndex == -1 {
		return strings.ToUpper(trimmed), ""
	}
	rtype = strings.ToUpper(trimmed[:splitIndex])
	payload = strings.TrimSpace(trimmed[splitIndex:])
	return rtype, payload
}

// DecodeLuaPayload normalizes the LUA payload for storage in RecordConfig.target.
func DecodeLuaPayload(payload string) (string, error) {
	if payload == "" {
		return "", nil
	}
	return txtutil.ParseQuoted(payload)
}

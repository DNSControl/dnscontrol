package models

import (
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

/*
For notes on how TXT records are handled, see documentation/developer-info/cookbook.md
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
// When .Type is "LUA", preserve its emitted record type while replacing the
// payload in its RDATA.
func (rc *RecordConfig) SetTargetTXT(s string) error {
	if rc.Type == "LUA" {
		rd := rc.AsLUA()
		rd.LuaPayload = s
		rc.SetRDATA(rd)
		return nil
	}
	return legacySetTargetArgsTXT(rc, s)
}

func legacySetTargetArgsTXT(rc *RecordConfig, args ...any) error {

	rc.TypeNum = dnsv2.TypeTXT
	rc.Type = "TXT"

	if rc.Metadata == nil {
		rc.Metadata = map[string]string{}
	}

	rd, err := MakeTXT("", nil, nrc.Flags{}, args...)
	if err != nil {
		log.Fatalf("legacySetTargetArgs: Failed to create RDATA for type %s: %+v", rc.Type, err)
	}
	rc.SetRDATA(rd)

	return nil
}

// SetTargetTXTs joins the supplied TXT fields and stores canonical 255-octet segments.
func (rc *RecordConfig) SetTargetTXTs(s []string) error {
	return rc.SetTargetTXT(strings.Join(s, ""))
}

// TXTJoined returns all the character-strings in a TXT RDATA as one string.
// FIXME(tlim): Unexport.
func TXTJoined(rd dnsrdatav2.TXT) string {
	return strings.Join(rd.Txt, "")
}

// TXTSegmented returns a TXT RDATA in DNSControl's canonical form: the
// character-strings are joined and then split into 255-octet segments. Empty
// input is represented by one empty segment.
// FIXME(tlim): Unexport.
func TXTSegmented(rd dnsrdatav2.TXT) []string {
	return splitChunks(TXTJoined(rd), 255)
}

// txtProperlySegmented returns true the TXT segments are properly segmented.
//   - There must be at least one segment.
//   - If there is one segment, it may be empty. It must be 255 octets or less.
//   - If there is more than 1 segment, all but the last must be exactly 255
//     octets. The last segment must be 255 octets or less but can not be empty.
func txtProperlySegmented(txts []string) bool {
	if len(txts) == 0 {
		return false
	}
	if len(txts) == 1 {
		return len(txts[0]) <= 255
	}
	for i := 0; i < len(txts)-1; i++ {
		if len(txts[i]) != 255 {
			return false
		}
	}
	last := txts[len(txts)-1]
	return len(last) > 0 && len(last) <= 255
}

// txtPayload returns the record's TXT payload as one string. It supports both
// TXT rdata and LUA rdata (whose payload is TXT-format).
func (rc *RecordConfig) txtPayload() string {
	switch rd := rc.GetRDATA().(type) {
	case dnsrdatav2.TXT:
		return TXTJoined(rd)
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
	if rd, ok := rc.rdata.(dnsrdatav2.TXT); ok {
		if !txtProperlySegmented(rd.Txt) {
			fmt.Fprintf(os.Stderr, "WARNING: GetTargetTXTSegmented: TXT record not properly segmented. Someone is not using SetRDATA? txt=%+v\n", rd.Txt)
			return splitChunks(rc.txtPayload(), 255)
		}
		return rd.Txt
	}
	return splitChunks(rc.txtPayload(), 255)
}

// GetTargetTXTSegmentCount returns the number of 255-octet segments required to store TXT target.
func (rc *RecordConfig) GetTargetTXTSegmentCount() int {
	return len(rc.GetTargetTXTSegmented())
}

func splitChunks(buf string, lim int) []string {
	if len(buf) == 0 {
		return []string{""}
	}
	if len(buf) <= lim {
		return []string{buf}
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

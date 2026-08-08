package rwth

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/dnsrr"
	"github.com/DNSControl/dnscontrol/v5/pkg/prettyzone"
)

// Print the generateZoneFileHelper.
func (api *rwthProvider) printRecConfig(rr models.RecordConfig) string {
	// Similar to prettyzone
	// Fake types are commented out.
	prefix := ""
	if _, err := dnsutilv2.StringToType(rr.Type); err != nil {
		prefix = ";"
	}

	// ttl
	ttl := ""
	if rr.TTL != 172800 && rr.TTL != 0 {
		ttl = strconv.FormatUint(uint64(rr.TTL), 10)
	}

	// type
	typeStr := rr.Type

	// the remaining line
	var target string
	if rr.GetRDATA() != nil {
		target = rr.GetRDATA().String()
	} else {
		panic("should not happen")
	}

	// comment
	comment := ";"

	return fmt.Sprintf("%s%s%s\n",
		prefix, prettyzone.FormatLine([]int{10, 5, 2, 5, 0}, []string{rr.NameFQDN, ttl, "IN", typeStr, target}), comment)
}

// toRecordConfig converts an RWTH record to a RecordConfig. It returns nil for
// the records that RWTH locks, which cannot be managed.
func toRecordConfig(dc *models.DomainConfig, apiRecord RecordReply) (*models.RecordConfig, error) {
	if checkIsLockedSystemAPIRecord(apiRecord) != nil {
		return nil, nil
	}

	dnsRec, err := NewRR(apiRecord.Content) // Parse content as DNS record
	if err != nil {
		return nil, err
	}

	recConfig, err := dnsrr.RRv2toRC(dc, dnsRec) // and make it a RC
	if err != nil {
		return nil, err
	}
	recConfig.Original = apiRecord // but keep our ApiRecord as the original

	return recConfig, nil
}

// NewRR returns custom dns.NewRR with RWTH default TTL.
func NewRR(s string) (dnsv2.RR, error) {
	if len(s) > 0 && s[len(s)-1] != '\n' { // We need a closing newline
		return ReadRR(strings.NewReader(s + "\n"))
	}
	return ReadRR(strings.NewReader(s))
}

// ReadRR reads an RR from r.
func ReadRR(r io.Reader) (dnsv2.RR, error) {
	zp := dnsv2.NewZoneParser(r, ".", "")
	zp.SetDefaultTTL(172800)
	rr, _ := zp.Next()
	return rr, zp.Err()
}

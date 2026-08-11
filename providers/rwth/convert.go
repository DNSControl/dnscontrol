package rwth

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/prettyzone"
	dnsv1 "github.com/miekg/dns"
)

// Print the generateZoneFileHelper.
func (api *rwthProvider) printRecConfig(rr models.RecordConfig) string {
	// Similar to prettyzone
	// Fake types are commented out.
	prefix := ""
	_, ok := dnsv1.StringToType[rr.Type]
	if !ok {
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
	target := rr.GetTargetCombined()

	// comment
	comment := ";"

	return fmt.Sprintf("%s%s%s\n",
		prefix, prettyzone.FormatLine([]int{10, 5, 2, 5, 0}, []string{rr.NameFQDN, ttl, "IN", typeStr, target}), comment)
}

// NewRR returns custom dns.NewRR with RWTH default TTL.
func NewRR(s string) (dnsv1.RR, error) {
	if len(s) > 0 && s[len(s)-1] != '\n' { // We need a closing newline
		return ReadRR(strings.NewReader(s + "\n"))
	}
	return ReadRR(strings.NewReader(s))
}

// ReadRR reads an RR from r.
func ReadRR(r io.Reader) (dnsv1.RR, error) {
	zp := dnsv1.NewZoneParser(r, ".", "")
	zp.SetDefaultTTL(172800)
	zp.SetIncludeAllowed(true)
	rr, _ := zp.Next()
	return rr, zp.Err()
}

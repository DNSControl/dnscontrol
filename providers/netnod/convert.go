package netnod

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	netnodPrimaryDNS "github.com/netnod/netnod-primary-dns-client"
)

// toRecordConfig converts a Netnod DNS Record to a RecordConfig. #rtype_variations.
func toRecordConfig(domain string, r netnodPrimaryDNS.Record, ttl int, name string, rtype string) (*models.RecordConfig, error) {
	// trimming trailing dot and domain from name
	name = strings.TrimSuffix(name, domain+".")
	name = strings.TrimSuffix(name, ".")

	rc := &models.RecordConfig{
		TTL:      uint32(ttl),
		Original: r,
		Type:     rtype,
	}
	rc.SetLabel(name, domain)

	switch rtype {
	case "TXT":
		// API accepts long TXTs without requiring to split them.
		// The API then returns them as they initially came in, e.g. "averylooooooo[...]oooooongstring" or "string" "string"
		// So we need to strip away " and split into multiple string
		// We can't use SetTargetRFC1035Quoted, it would split the long strings into multiple parts
		return rc, rc.SetTargetTXTs(parseTxt(r.Content))
	default:
		return rc, rc.PopulateFromString(rtype, r.Content, domain)
	}
}

func parseTxt(content string) (result []string) {
	for r := range strings.SplitSeq(content, "\" ") {
		result = append(result, strings.Trim(r, "\""))
	}
	return
}

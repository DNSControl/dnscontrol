package netnod

import (
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	netnodPrimaryDNS "github.com/netnod/netnod-primary-dns-client"
)

// toRecordConfig converts a Netnod DNS Record to a RecordConfig. #rtype_variations.
func toRecordConfig(dc *models.DomainConfig, r netnodPrimaryDNS.Record, ttl int, name string, rtype string) (*models.RecordConfig, error) {
	label := dc.LabelFromShort(name)
	if strings.HasSuffix(name, ".") {
		label = dc.LabelFromFQDNWithDot(name)
	}
	var rc *models.RecordConfig
	var err error
	switch rtype {
	case "TXT":
		// API accepts long TXTs without requiring to split them.
		// The API then returns them as they initially came in, e.g. "averylooooooo[...]oooooongstring" or "string" "string"
		// So we need to strip away " and split into multiple string
		// We can't use SetTargetRFC1035Quoted, it would split the long strings into multiple parts
		rc, err = dc.NewRecordConfig(label, uint32(ttl), dnsv2.TypeTXT, strings.Join(parseTxt(r.Content), ""))
	default:
		rc, err = dc.NewRecordConfigParse(label, uint32(ttl), rtype, r.Content)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r
	return rc, nil
}

func parseTxt(content string) (result []string) {
	for r := range strings.SplitSeq(content, "\" ") {
		result = append(result, strings.Trim(r, "\""))
	}
	return
}

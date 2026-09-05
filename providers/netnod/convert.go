package netnod

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	netnodPrimaryDNS "github.com/netnod/netnod-primary-dns-client"
)

// toRecordConfig converts a Netnod DNS Record to a RecordConfig. #rtype_variations.
func toRecordConfig(dc *models.DomainConfig, r netnodPrimaryDNS.Record, ttl uint32, name string, rtype string) (*models.RecordConfig, error) {

	// API accepts long TXTs without requiring to split them.
	// The API then returns them as they initially came in, e.g. "averylooooooo[...]oooooongstring" or "string" "string"
	// This means the first time we see a TXT record we might rewrite it as 255+255+255+remainder.

	rc, err := dc.NewRecordConfigParse(dc.LabelFromFQDNWithDot(name), ttl, rtype, r.Content)
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

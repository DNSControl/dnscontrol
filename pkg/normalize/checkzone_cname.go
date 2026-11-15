package normalize

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func checkCNAMEs(dc *models.DomainConfig, extra func(*models.RecordConfig) string) (errs []error) {
	cnames := map[string]bool{}
	proxiedCnames := map[string]bool{}
	seenIdentity := map[string]bool{}
	for _, r := range dc.Records {
		if r.Type == "CNAME" {
			// Without a provider-declared identity this is exactly the old
			// rule: one CNAME per label. With one, two CNAMEs may share a
			// label as long as the provider treats them as separate objects.
			id := r.GetLabel()
			if x := providerIdentity(r, extra); x != "" {
				id += "|" + x
			}
			if seenIdentity[id] {
				errs = append(errs, fmt.Errorf("%s: cannot have multiple CNAMEs with same name: %s", r.FilePos, r.GetLabelFQDN()))
			}
			seenIdentity[id] = true
			cnames[r.GetLabel()] = true
			if p, ok := r.Metadata["cloudflare_proxy"]; ok && (p == "on" || p == "full") {
				proxiedCnames[r.GetLabel()] = true
			}
		}
	}
	for _, r := range dc.Records {
		if cnames[r.GetLabel()] && r.Type != "CNAME" {
			// Allow AKAMAICDN and CNAME to have same name
			if r.Type == "AKAMAICDN" {
				continue
			}
			// Cloudflare proxied (flattened) CNAMEs are resolved internally
			// and never served as actual CNAME records, so the RFC 1034 §3.6.2
			// restriction does not apply.
			if proxiedCnames[r.GetLabel()] {
				continue
			}
			errs = append(errs, fmt.Errorf("%s: cannot have CNAME and %s record with same name: %s", r.FilePos, r.Type, r.GetLabelFQDN()))
		}
	}
	return
}

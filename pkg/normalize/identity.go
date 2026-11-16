package normalize

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

// recordIdentityString returns the string used to detect duplicate records.
// It is the label, the rType, the RDATA and any provider-declared identity
// text (for example DNSPod's record line).
func recordIdentityString(r *models.RecordConfig, extra func(*models.RecordConfig) string) string {
	id := fmt.Sprintf("%s %s %s", r.GetLabelFQDN(), r.Type, r.ComparableV3)
	if x := providerIdentity(r, extra); x != "" {
		id += " " + x
	}
	return id
}

// domainRecordIdentity returns the identity function declared by one of the
// domain's DNS providers, or nil if none declares one.
func domainRecordIdentity(d *models.DomainConfig) func(*models.RecordConfig) string {
	for _, provider := range d.DNSProviderInstances {
		if provider.ProviderType == "-" {
			continue
		}
		if f := providers.GetRecordIdentity(provider.ProviderType); f != nil {
			return f
		}
	}
	return nil
}

// providerIdentity returns the provider-declared identity text for a record,
// or "" when no provider declares one.
func providerIdentity(r *models.RecordConfig, extra func(*models.RecordConfig) string) string {
	if extra == nil {
		return ""
	}
	return extra(r)
}

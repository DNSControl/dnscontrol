package normalize

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func checkProviderCapabilities(dc *models.DomainConfig) error {
	// Record types are checked per record, by validateSupportedRecordTypes.
	// AUTODNSSEC is not a record type, so SupportedTypes does not cover it.
	if dc.AutoDNSSEC == "" {
		return nil
	}
	for _, provider := range dc.DNSProviderInstances {
		if provider.ProviderType == "-" {
			// "-" indicates that we don't yet know who the provider type
			// is.  This is probably due to the fact that `dnscontrol
			// check` doesn't read creds.json, which is where the TYPE is
			// set.  We will skip this test in this instance.  Later if
			// `dnscontrol preview` or `push` is used, the full check will
			// be performed.
			continue
		}
		def, ok := providers.GetDefinition(provider.ProviderType)
		if !ok || !def.CanAutoDNSSEC.HasFeature {
			return fmt.Errorf("domain %s uses AUTODNSSEC records, but DNS provider type %s does not support them", dc.Name, provider.ProviderType)
		}
	}
	return nil
}

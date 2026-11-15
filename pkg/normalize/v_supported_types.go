package normalize

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

// validateSupportedRecordTypes checks catalog recognition and provider support
// for original types before transformations and resulting types afterwards.
func validateSupportedRecordTypes(rec *models.RecordConfig, domain string, pTypes []string) error {
	if _, known := privatetypes.LookupRecordType(rec.Type); !known {
		return fmt.Errorf("unknown record type %s in domain %s", rec.Type, domain)
	}
	// rec.TypeNum may be stale after transformations, so derive it from rec.Type.
	typeNum := dnsv2.StringToType[rec.Type]
	for _, pType := range pTypes {
		if providers.RecordTypeSupported(pType, typeNum, rec.GetLabel()) {
			continue
		}
		return fmt.Errorf("domain %s uses %s records, but DNS provider type %s does not support them at %s", domain, rec.Type, pType, rec.GetLabel())
	}
	return nil
}

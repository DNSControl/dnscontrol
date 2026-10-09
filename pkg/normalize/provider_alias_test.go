package normalize

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func init() {
	providers.Register[*validationProvider]("TEST_CUSTOM_PROVIDER", providers.Definition{
		FriendlyName: "Custom type test", Aliases: []string{"TEST_CUSTOM_ALIAS"},
		SupportedTypes: []string{
			"R53_ALIAS",
			"ALIAS",
		},
	})
}

func TestCustomRecordProviderAliases(t *testing.T) {
	for _, rtype := range []string{"R53_ALIAS", "ALIAS"} {
		for _, providerType := range []string{"TEST_CUSTOM_PROVIDER", "TEST_CUSTOM_ALIAS"} {
			r := &models.RecordConfig{Type: rtype, Metadata: map[string]string{}}
			if err := validateSupportedRecordTypes(r, "example.com", []string{providerType}); err != nil {
				t.Fatal(err)
			}
			if r.Type != rtype {
				t.Fatal("validation changed the record type")
			}
			if err := validateSupportedRecordTypes(r, "example.com", []string{providerType, "S5_BASIC8"}); err == nil {
				t.Fatal("custom type accepted by an unrelated provider")
			}
		}
	}
}

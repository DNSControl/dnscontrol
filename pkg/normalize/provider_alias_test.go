package normalize

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func init() {
	providers.Register[*validationProvider]("TEST_CUSTOM_PROVIDER", providers.Definition{
		FriendlyName: "Custom type test", Aliases: []string{"TEST_CUSTOM_ALIAS"},
		Features: providers.DocumentationNotes{}, // Exercise legacy ownership checks.
	})
	providers.RegisterCustomRecordType("TEST_CUSTOM_TYPE", "TEST_CUSTOM_PROVIDER", "")
	providers.RegisterCustomRecordType("TEST_CUSTOM_TYPE_ALIAS_OWNER", "TEST_CUSTOM_ALIAS", "")
}

func TestCustomRecordProviderAliases(t *testing.T) {
	for _, rtype := range []string{"TEST_CUSTOM_TYPE", "TEST_CUSTOM_TYPE_ALIAS_OWNER"} {
		for _, providerType := range []string{"TEST_CUSTOM_PROVIDER", "TEST_CUSTOM_ALIAS"} {
			r := &models.RecordConfig{Type: rtype, Metadata: map[string]string{}}
			if err := validateLegacyRecordTypes(r, "example.com", []string{providerType}); err != nil {
				t.Fatal(err)
			}
			if r.Metadata["orig_custom_type"] != rtype {
				t.Fatal("custom-type validation marker was lost")
			}
			if err := validateLegacyRecordTypes(r, "example.com", []string{providerType, plainProviderType}); err == nil {
				t.Fatal("custom type accepted by an unrelated provider")
			}
		}
	}
}

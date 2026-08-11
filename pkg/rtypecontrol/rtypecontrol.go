package rtypecontrol

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

// RType is an interface that defines the methods required for a DNS record type.
type RType interface {
	// Returns the name of the rtype ("A", "MX", etc.)
	Name() string

	// RecordConfig factory. Updates a RecordConfig's fields based on args.
	FromArgs(*domaintags.DomainNameVarieties, *models.RecordConfig, []any) error
	FromStruct(*domaintags.DomainNameVarieties, *models.RecordConfig, string, any) error

	CopyToLegacyFields(*models.RecordConfig)
	CopyFromLegacyFields(*models.RecordConfig)
}

// Func is a map of registered rtypes.
var Func map[string]RType = map[string]RType{}

// Register registers a new RType (Record Type) implementation. It can be used
// to register an RFC-defined type, a new custom type, or a "builder".
//
// It panics if the type is already registered, to prevent accidental overwrites.
func Register(t RType) {
	name := t.Name()
	if _, ok := Func[name]; ok {
		panic(fmt.Sprintf("rtype %q already registered. Can't register it a second time!", name))
	}
	// Store the interface
	Func[name] = t

	// For compatibility with legacy systems:
	providers.RegisterCustomRecordType(name, "", "")
}

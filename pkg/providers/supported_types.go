package providers

import (
	"fmt"
	"maps"
	"regexp"
	"strings"
	"sync"

	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
)

// RecordTypeCapabilities bridges SupportedTypes and legacy feature consumers.
// Treat this map as read-only. Operational capabilities do not belong here.
var RecordTypeCapabilities = map[string]Capability{
	"AKAMAICDN": CanUseAKAMAICDN, "AKAMAITLC": CanUseAKAMAITLC,
	"ALIAS": CanUseAlias, "AZURE_ALIAS": CanUseAzureAlias,
	"CAA": CanUseCAA, "DHCID": CanUseDHCID, "DNAME": CanUseDNAME,
	"DNSKEY": CanUseDNSKEY, "DS": CanUseDS, "HTTPS": CanUseHTTPS,
	"LOC": CanUseLOC, "NAPTR": CanUseNAPTR, "OPENPGPKEY": CanUseOPENPGPKEY,
	"PTR": CanUsePTR, "R53_ALIAS": CanUseRoute53Alias, "RP": CanUseRP,
	"SMIMEA": CanUseSMIMEA, "SOA": CanUseSOA, "SRV": CanUseSRV,
	"SSHFP": CanUseSSHFP, "SVCB": CanUseSVCB, "TLSA": CanUseTLSA,
}

type typeSelector struct {
	source   string
	name     string
	pattern  *regexp.Regexp
	priority int
	note     DocumentationNote
}

func (d *Definition) compileTypeSelectors() error {
	selectors := d.SupportedTypes
	if selectors == nil && d.Features == nil {
		selectors = []string{"Default"}
	}
	for _, source := range selectors {
		name, status, suffixed := strings.Cut(source, ":")
		name = strings.ToUpper(name)
		s := typeSelector{source: source, name: name, priority: 4, note: DocumentationNote{HasFeature: true}}
		if name == "" || strings.Trim(name, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_*") != "" {
			return fmt.Errorf("provider %q: malformed SupportedTypes selector %q", d.TypeName, source)
		}
		if suffixed {
			switch status {
			case "Can":
			case "Cannot":
				s.note.HasFeature = false
			case "Unimplemented":
				s.note.HasFeature, s.note.Unimplemented = false, true
			default:
				return fmt.Errorf("provider %q: invalid SupportedTypes status in %q", d.TypeName, source)
			}
		}
		switch {
		case name == "DEFAULT" || name == "RFC":
			if suffixed {
				return fmt.Errorf("provider %q: category %q cannot have a status suffix", d.TypeName, source)
			}
		case strings.Contains(name, "*"):
			// Only '*' is special: it matches zero or more characters in a
			// whole type name. Other glob syntax is intentionally rejected.
			s.pattern = regexp.MustCompile("^" + strings.ReplaceAll(regexp.QuoteMeta(name), `\*`, ".*") + "$")
			if suffixed {
				s.priority = 2
			}
		default:
			s.priority = 1
		}
		d.typeSelectors = append(d.typeSelectors, s)
	}
	return nil
}

func (s typeSelector) matches(typ privatetypes.RecordType) bool {
	switch s.name {
	case "DEFAULT":
		switch typ.Name {
		case "A", "AAAA", "CAA", "CNAME", "MX", "NS", "SRV", "TXT":
			return true
		}
		return false
	case "RFC":
		return !typ.Pseudo
	}
	if s.pattern != nil {
		return s.pattern.MatchString(typ.Name)
	}
	return s.name == typ.Name
}

func sameTypeStatus(a, b DocumentationNote) bool {
	return a.HasFeature == b.HasFeature && a.Unimplemented == b.Unimplemented
}

func (d *Definition) resolveType(typ privatetypes.RecordType) (DocumentationNote, error) {
	priority := 5
	var result DocumentationNote
	var legacy *DocumentationNote
	if capability, ok := RecordTypeCapabilities[typ.Name]; ok {
		legacy = d.Features[capability]
		if legacy != nil {
			priority, result = 3, *legacy
		}
	}
	// Find the winning priority first. An exact declaration may resolve
	// conflicts between lower-priority patterns, regardless of input order.
	for _, s := range d.typeSelectors {
		if s.priority < priority && s.matches(typ) {
			priority = s.priority
		}
	}
	var previous string
	for _, s := range d.typeSelectors {
		if s.priority != priority || !s.matches(typ) {
			continue
		}
		if previous != "" && !sameTypeStatus(result, s.note) {
			return DocumentationNote{}, fmt.Errorf("provider %q: conflicting SupportedTypes for %s: %q and %q", d.TypeName, typ.Name, previous, s.source)
		}
		previous, result = s.source, s.note
	}
	if legacy != nil && sameTypeStatus(result, *legacy) {
		result = *legacy
	}
	return result, nil
}

// UsesSupportedTypes reports whether validation must check every record type.
// Legacy Features without SupportedTypes retains the old validation coverage.
func (d *Definition) UsesSupportedTypes() bool { return d.exhaustiveTypes }

// RecordTypeSupport returns the effective status and compatible legacy notes.
// Unknown types are unsupported, including when '*' is declared. In legacy
// mode this describes only explicit capabilities, not an exhaustive inventory.
func (d *Definition) RecordTypeSupport(name string) DocumentationNote {
	typ, ok := privatetypes.LookupRecordType(strings.ToUpper(name))
	if !ok {
		return DocumentationNote{}
	}
	note, err := d.resolveType(typ)
	if err != nil {
		panic(err) // Finalize detects conflicts before consumers read definitions.
	}
	return note
}

var (
	finalizeMu           sync.Mutex
	definitionGeneration uint64
	finalizedGeneration  uint64
	finalizedCatalog     uint64
)

// Finalize validates selectors against the complete record catalog and derives
// compatibility capabilities. Call after package registration and before
// concurrent reads. The _all package does this at startup; accessors also ensure
// finalization for programs/tests importing individual providers.
//
// Register and private-type registration invalidate the result. Tests adding
// either must finish registration before reading again. Repeated calls without
// registrations do not mutate definitions. Registration concurrent with reads
// remains unsupported.
func Finalize() error {
	finalizeMu.Lock()
	defer finalizeMu.Unlock()
	catalogVersion := privatetypes.CatalogVersion()
	if finalizedGeneration == definitionGeneration && finalizedCatalog == catalogVersion {
		return nil
	}
	catalog := privatetypes.RecordTypes()
	updates := map[*Definition]DocumentationNotes{}
	for _, d := range allDefinitions() {
		if !d.exhaustiveTypes {
			continue
		}
		for _, s := range d.typeSelectors {
			if s.priority == 1 {
				if _, ok := privatetypes.LookupRecordType(s.name); !ok {
					return fmt.Errorf("provider %q: unknown record type in SupportedTypes selector %q", d.TypeName, s.source)
				}
			}
		}
		features := maps.Clone(d.DerivedFeatures)
		for _, typ := range catalog {
			note, err := d.resolveType(typ)
			if err != nil {
				return err
			}
			if capability, ok := RecordTypeCapabilities[typ.Name]; ok {
				features[capability] = &note
			}
		}
		// General DS support implies child support. Rebuild from the original
		// inputs so later catalog registrations never retain stale notes.
		child := d.CanUseDSForChildren
		if child == nil {
			child = d.Features[CanUseDSForChildren]
		}
		if features[CanUseDS].HasFeature {
			if child == nil || !child.HasFeature {
				child = Can()
			}
		}
		if child != nil {
			features[CanUseDSForChildren] = child
		}
		updates[d] = features
	}
	// Publish only after every declaration is valid.
	for d, features := range updates {
		d.DerivedFeatures = features
	}
	finalizedGeneration, finalizedCatalog = definitionGeneration, catalogVersion
	return nil
}

func mustFinalize() {
	if err := Finalize(); err != nil {
		panic(err)
	}
}

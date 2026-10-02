//go:generate go tool stringer -type=Capability

package providers

import (
	"log"
)

// Capability is a bitmasked set of "features" that a provider supports. Only use constants from this package.
type Capability int

const (
	// Keep this list sorted.
	// If you add something here, you probably want to also add it to
	// pkg/normalize/validate.go checkProviderCapabilities() or
	// somewhere near there.

	// CanAutoDNSSEC indicates that the provider can automatically handle DNSSEC,
	// so folks can ask for that.
	CanAutoDNSSEC Capability = iota

	// CanConcur indicates the provider can be used concurrently to gather zone data.
	// Can() indicates that it has been tested and shown to work concurrently.
	// Cannot() indicates it has not been tested OR it has been shown to not
	// work when used concurrently.  The default is Cannot().
	CanConcur

	// CanGetZones indicates the provider can enumerate zones (get-zones ... all).
	// Reading explicitly named zones does not require this capability.
	CanGetZones

	// CanOnlyDiff1Features indicates the provider has not yet been upgraded to
	// use the "diff2" differencing engine.  Instead, it uses the the backwards
	// compatibility mode.  The diff2 engine is required to repliably provide
	// IGNORE(), NO_PURGE, and other features.
	// This capability is set automatically for the provider during the call to
	// RegisterDomainServiceProviderType.  It is set to Can() if we detect
	// compatibility mode is in use. All other values (Unimplemented and Cannot)
	// are equivalent.
	CanOnlyDiff1Features

	// CanUseAKAMAICDN indicates the provider support the specific AKAMAICDN records that only the Akamai EdgeDns provider supports.
	CanUseAKAMAICDN

	// CanUseAlias indicates the provider support ALIAS records (or flattened CNAMES). Up to the provider to translate them to the appropriate record type.
	CanUseAlias

	// CanUseAzureAlias indicates the provider support the specific Azure_ALIAS records that only the Azure provider supports.
	CanUseAzureAlias

	// CanUseCAA indicates the provider can handle CAA records.
	CanUseCAA

	// CanUseDHCID indicates the provider can handle DHCID records.
	CanUseDHCID

	// CanUseDNAME indicates the provider can handle DNAME records.
	CanUseDNAME

	// CanUseDS indicates that the provider can handle DS record types. This
	// implies CanUseDSForChildren without specifying the latter explicitly.
	CanUseDS

	// CanUseDSForChildren indicates the provider can handle DS record types, but
	// only for children records, not at the root of the zone.
	CanUseDSForChildren

	// CanUseHTTPS indicates the provider can handle HTTPS records.
	CanUseHTTPS

	// CanUseLOC indicates whether service provider handles LOC records.
	CanUseLOC

	// CanUseNAPTR indicates the provider can handle NAPTR records.
	CanUseNAPTR

	// CanUsePTR indicates the provider can handle PTR records.
	CanUsePTR

	// CanUseRoute53Alias indicates the provider support the specific R53_ALIAS records that only the Route53 provider supports.
	CanUseRoute53Alias

	// CanUseRP indicates the provider can handle RP records.
	CanUseRP

	// CanUseSMIMEA indicates the provider can handle SMIMEA records.
	CanUseSMIMEA

	// CanUseSOA indicates the provider supports full management of a zone's SOA record.
	CanUseSOA

	// CanUseSRV indicates the provider can handle SRV records.
	CanUseSRV

	// CanUseSSHFP indicates the provider can handle SSHFP records.
	CanUseSSHFP

	// CanUseSVCB indicates the provider can handle SVCB records.
	CanUseSVCB

	// CanUseTLSA indicates the provider can handle TLSA records.
	CanUseTLSA

	// CanUseDNSKEY indicates that the provider can handle DNSKEY records.
	CanUseDNSKEY

	// CanUseOPENPGPKEY indicates that the provider can handle OPENPGPKEY records.
	CanUseOPENPGPKEY

	// DocCreateDomains means provider can add domains with the `dnscontrol create-domains` command.
	DocCreateDomains

	// DocDualHost means provider allows full management of apex NS records, so we can safely dual-host with another provider.
	DocDualHost

	// DocOfficiallySupported means it is actively used and maintained by stack exchange.
	DocOfficiallySupported

	// CanUseAKAMAITLC indicates the provider supports the specific AKAMAITLC records that only the Akamai EdgeDns provider supports.
	CanUseAKAMAITLC
)

var providerCapabilities = map[string]map[Capability]bool{}

// ProviderHasCapability returns true if provider has capability.
func ProviderHasCapability(pType string, capa Capability) bool {
	if def, ok := definitions[pType]; ok {
		note := def.DerivedFeatures[capa]
		return note != nil && note.HasFeature
	}
	if providerCapabilities[pType] == nil {
		return false
	}
	return providerCapabilities[pType][capa]
}

// DocumentationNote is a way for providers to give more detail about what features they support.
type DocumentationNote struct {
	HasFeature    bool
	Unimplemented bool
	Comment       string
	Link          string
}

// DocumentationNotes is a full list of notes for a single provider.
type DocumentationNotes map[Capability]*DocumentationNote

// ProviderMetadata is a common interface for DocumentationNotes and Capability to be used interchangeably.
type ProviderMetadata any

// Notes is a collection of all documentation notes, keyed by provider type.
var Notes = map[string]DocumentationNotes{}

func unwrapProviderCapabilities(pName string, meta []ProviderMetadata) {
	if providerCapabilities[pName] == nil {
		providerCapabilities[pName] = map[Capability]bool{}
	}
	for _, pm := range meta {
		switch x := pm.(type) {
		case Capability:
			providerCapabilities[pName][x] = true
		case DocumentationNotes:
			if Notes[pName] == nil {
				Notes[pName] = DocumentationNotes{}
			}
			for k, v := range x {
				Notes[pName][k] = v
				providerCapabilities[pName][k] = v.HasFeature
			}
		default:
			log.Fatalf("Unrecognized ProviderMetadata type: %T", pm)
		}
	}
}

// Can is a small helper for concisely creating Documentation Notes
// comments are variadic for easy omission. First is comment, second is link, the rest are ignored.
func Can(comments ...string) *DocumentationNote {
	n := &DocumentationNote{
		HasFeature: true,
	}
	n.addStrings(comments)
	return n
}

// Cannot is a small helper for concisely creating Documentation Notes
// comments are variadic for easy omission. First is comment, second is link, the rest are ignored.
func Cannot(comments ...string) *DocumentationNote {
	n := &DocumentationNote{
		HasFeature: false,
	}
	n.addStrings(comments)
	return n
}

// Unimplemented is a small helper for concisely creating Documentation Notes
// comments are variadic for easy omission. First is comment, second is link, the rest are ignored.
func Unimplemented(comments ...string) *DocumentationNote {
	n := &DocumentationNote{
		HasFeature:    false,
		Unimplemented: true,
	}
	n.addStrings(comments)
	return n
}

func (n *DocumentationNote) addStrings(comments []string) {
	if len(comments) > 0 {
		n.Comment = comments[0]
	}
	if len(comments) > 1 {
		n.Link = comments[1]
	}
}

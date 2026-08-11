package rtype

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v4/pkg/rtypecontrol"
	dnsv1 "github.com/miekg/dns"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
)

func init() {
	rtypecontrol.Register(&RP{})
}

// RP RR. See RFC 1138, Section 2.2.
type RP struct {
	dnsv1.RP
}

// Name returns the DNS record type as a string.
func (handle *RP) Name() string {
	return "RP"
}

// FromArgs fills in the RecordConfig from []any, which is typically from a parsed config file.
func (handle *RP) FromArgs(dcn *domaintags.DomainNameVarieties, rec *models.RecordConfig, args []any) error {
	if err := rtypecontrol.PaveArgs(args[1:], "ss"); err != nil {
		return fmt.Errorf("ERROR: (%s) [RP(%q, %v)]: %w",
			rec.FilePos,
			rec.Name, rtypecontrol.StringifyQuoted(args[1:]),
			err)
	}
	fields := &RP{
		dnsv1.RP{
			Mbox: dnsutilv1.AddOrigin(args[1].(string), dcn.NameASCII+"."),
			Txt:  dnsutilv1.AddOrigin(args[2].(string), dcn.NameASCII+"."),
		},
	}

	return handle.FromStruct(dcn, rec, args[0].(string), fields)
}

// FromStruct fills in the RecordConfig from a struct, typically from an API response.
func (handle *RP) FromStruct(dcn *domaintags.DomainNameVarieties, rec *models.RecordConfig, name string, fields any) error {
	rec.F = fields

	rec.ZonefilePartial = rec.GetTargetRFC1035Quoted()
	rec.Comparable = rec.ZonefilePartial

	handle.CopyToLegacyFields(rec)
	return nil
}

// CopyToLegacyFields populates the legacy fields of the RecordConfig using the fields in .F.
func (handle *RP) CopyToLegacyFields(rec *models.RecordConfig) {
	// RP, like all new RRs, does not have legacy fields. Even .target is deprecated.
}

// CopyFromLegacyFields populates the legacy fields of the RecordConfig using the fields in .F.
func (handle *RP) CopyFromLegacyFields(rec *models.RecordConfig) {
	// RP is RecordConfigv2 and has no legacy fields. Even .target is deprecated.

	// Fix up ZonefilePartial and Comparable:
	rec.ZonefilePartial = rec.GetTargetRFC1035Quoted()
	rec.Comparable = rec.ZonefilePartial
}

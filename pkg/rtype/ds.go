package rtype

// import (
// 	"fmt"

// 	dnsv2 "codeberg.org/miekg/dns"
// 	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
// 	"github.com/DNSControl/dnscontrol/v4/models"
// 	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
// 	"github.com/DNSControl/dnscontrol/v4/pkg/rtypecontrol"
// 	dnsv1 "github.com/miekg/dns"
// )

// func init() {
// 	rtypecontrol.Register(&DS{})
// }

// // DS RR.
// type DS struct {
// 	dnsrdatav2.DS
// }

// // Name returns the DNS record type as a string.
// func (handle *DS) Name() string {
// 	return "DS"
// }

// // FromArgs fills in the RecordConfig from []any, which is typically from a parsed config file.
// func (handle *DS) FromArgs(dcn *domaintags.DomainNameVarieties, rec *models.RecordConfig, args []any) error {
// 	if err := rtypecontrol.PaveArgs(args[1:], "wbbs"); err != nil {
// 		return fmt.Errorf("ERROR: (%s) [DS(%q, %v)]: %w",
// 			rec.FilePos,
// 			rec.Name, rtypecontrol.StringifyQuoted(args[1:]),
// 			err)
// 	}
// 	fields := &dnsrdatav2.DS{
// 		KeyTag:     args[1].(uint16),
// 		Algorithm:  args[2].(uint8),
// 		DigestType: args[3].(uint8),
// 		Digest:     args[4].(string),
// 	}

// 	return handle.FromStruct(dcn, rec, args[0].(string), fields)
// }

// // FromStruct fills in the RecordConfig from a struct, typically from an API response.
// func (handle *DS) FromStruct(dcn *domaintags.DomainNameVarieties, rec *models.RecordConfig, name string, fields any) error {
// 	// Fields is of type "any" thus we must validate the type. It should be the "inner" type of .F, not the outer type, rtype.DS{}.

// 	// This is a temporary hack to deal with the fact that the code is in transition from dnsv1 to dnsv2.
// 	if ds, ok := fields.(dnsrdatav2.DS); ok {
// 		rec.F = &DS{ds}
// 	} else if ds, ok := fields.(*dnsrdatav2.DS); ok {
// 		rec.F = &DS{*ds}
// 	} else if ds, ok1 := fields.(*dnsv1.DS); ok1 {
// 		rec.F = &DS{
// 			dnsrdatav2.DS{
// 				KeyTag:     ds.KeyTag,
// 				Algorithm:  ds.Algorithm,
// 				DigestType: ds.DigestType,
// 				Digest:     ds.Digest,
// 			},
// 		}
// 	} else if ds, ok2 := fields.(*dnsv2.DS); ok2 {
// 		rec.F = &DS{
// 			dnsrdatav2.DS{
// 				KeyTag:     ds.KeyTag,
// 				Algorithm:  ds.Algorithm,
// 				DigestType: ds.DigestType,
// 				Digest:     ds.Digest,
// 			},
// 		}
// 	} else {
// 		panic(fmt.Sprintf("assertion failed: fields not rdatav2, dns1 or dns2 .DS, got %T", fields))
// 		//return fmt.Errorf("fields is not *dns.DS, got %T", fields)
// 	}

// 	// Hack to deal with the fact that fixlegacy.go can't import rtype.
// 	switch rec.F.(type) {
// 	case *DS:
// 		rd, err := models.MakeDS(name, rec.F.(*DS).KeyTag, rec.F.(*DS).Algorithm, rec.F.(*DS).DigestType, rec.F.(*DS).Digest)
// 		if err != nil {
// 			return fmt.Errorf("MakeDS failed: %w", err)
// 		}
// 		rec.RDATA = rd
// 	case *dnsv1.DS:
// 		rd, err := models.MakeDS(name, rec.F.(*DS).KeyTag, rec.F.(*DS).Algorithm, rec.F.(*DS).DigestType, rec.F.(*DS).Digest)
// 		if err != nil {
// 			return fmt.Errorf("MakeDS failed: %w", err)
// 		}
// 		rec.RDATA = rd
// 	default:
// 		panic(fmt.Sprintf("unexpected type for DS.FromStruct: %T", rec.F))
// 	}
// 	rec.ComparableV3 = rec.RDATA.String()

// 	rec.ZonefilePartial = rec.GetTargetRFC1035Quoted()
// 	rec.Comparable = rec.ZonefilePartial

// 	handle.CopyToLegacyFields(rec)
// 	return nil
// }

// // CopyToLegacyFields populates the legacy fields of the RecordConfig using the fields in .F.
// func (handle *DS) CopyToLegacyFields(rec *models.RecordConfig) {
// 	ds := rec.F.(*DS)
// 	_ = rec.SetTargetDS(ds.KeyTag, ds.Algorithm, ds.DigestType, ds.Digest)
// }

// // CopyFromLegacyFields uses the the legacy fields to populate .F.
// func (handle *DS) CopyFromLegacyFields(rec *models.RecordConfig) {
// 	// Copy fields:
// 	rec.F = &DS{
// 		dnsrdatav2.DS{
// 			KeyTag:     rec.DsKeyTag,
// 			Algorithm:  rec.DsAlgorithm,
// 			DigestType: rec.DsDigestType,
// 			Digest:     rec.DsDigest,
// 		},
// 	}

// 	// Fix up ZonefilePartial and Comparable:
// 	rec.ZonefilePartial = rec.GetTargetRFC1035Quoted()
// 	rec.Comparable = rec.ZonefilePartial
// }

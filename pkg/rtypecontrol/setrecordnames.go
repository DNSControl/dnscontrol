package rtypecontrol

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
)

// This code defines many variables to make the logic easier to read. The Go optimizer
// should eliminate any performance impact.
// We could probably fold some of the logic together, but it would be harder to read.
// It's difficult enough to understand it as-is, so clarity is preferred.

// setRecordNames uses n to update the .Name* fields.  If the name is a FQDN
// (ends with a "."), it will be handled accordingly.  However if it does not
// match the domain name, no error is returned but rec.Name* fields will end
// with a ".".
func setRecordNames(rec *models.RecordConfig, dcn *domaintags.DomainNameVarieties, n string) error {
	if rec.SubDomain == "" {
		return setRecordNamesNonExtend(rec, dcn, n)
	}
	return setRecordNamesExtend(rec, dcn, n)
}

func setRecordNamesNonExtend(rec *models.RecordConfig, dcn *domaintags.DomainNameVarieties, n string) error {

	nRaw := n
	nASCII := domaintags.EfficientToASCII(n)
	nUnicode := domaintags.EfficientToUnicode(nASCII)

	if strings.HasSuffix(nRaw, ".") {
		// The user specified a FQDN, we give it special handling.

		// with trailing dot:
		nameRawdot := nRaw
		nameASCIIdot := nASCII
		nameUnicodedot := nUnicode
		// without trailing dot:
		nameRaw := nameRawdot[:len(nameRawdot)-1]
		nameASCII := nameASCIIdot[:len(nameASCIIdot)-1]
		nameUnicode := nameUnicodedot[:len(nameUnicodedot)-1]
		// suffix:
		suffixRaw := dcn.NameRaw
		suffixASCII := dcn.NameASCII
		suffixUnicode := dcn.NameUnicode
		dotsuffixASCII := "." + suffixASCII

		if nameASCII == suffixASCII {
			// Name is the apex:
			rec.NameRaw = "@"
			rec.Name = "@"
			rec.NameUnicode = "@"
			rec.NameFQDNRaw = dcn.NameRaw
			rec.NameFQDN = dcn.NameASCII
			rec.NameFQDNUnicode = dcn.NameUnicode
			return nil
		} else if strings.HasSuffix(nameASCII, dotsuffixASCII) {
			// Name is a subdomain of the domain:
			rec.NameRaw = nameRaw[:len(nameRaw)-len(suffixRaw)-1]
			rec.Name = nameASCII[:len(nameASCII)-len(suffixASCII)-1]
			rec.NameUnicode = nameUnicode[:len(nameUnicode)-len(suffixUnicode)-1]
			rec.NameFQDNRaw = nameRaw
			rec.NameFQDN = nameASCII
			rec.NameFQDNUnicode = nameUnicode
			return nil
		}
		// Name is not in this domain. Don't error, just return names with trailing dots.
		rec.NameRaw = nameRawdot
		rec.Name = nameASCIIdot
		rec.NameUnicode = nameUnicodedot
		rec.NameFQDNRaw = nameRawdot + dcn.NameRaw + "."
		rec.NameFQDN = nameASCIIdot + dcn.NameASCII + "."
		rec.NameFQDNUnicode = nameUnicodedot + dcn.NameUnicode + "."
		return nil
	}

	// Non-FQDN case:

	if nRaw == "@" {
		// Name is the apex. We never store "", we store "@".
		rec.NameRaw = "@"
		rec.Name = "@"
		rec.NameUnicode = "@"
		rec.NameFQDNRaw = dcn.NameRaw
		rec.NameFQDN = dcn.NameASCII
		rec.NameFQDNUnicode = dcn.NameUnicode
		return nil
	}

	// Everything else:

	nameRaw := nRaw
	nameASCII := nASCII
	nameUnicode := nUnicode

	rec.NameRaw = nameRaw
	rec.Name = nameASCII
	rec.NameUnicode = nameUnicode
	rec.NameFQDNRaw = nameRaw + "." + dcn.NameRaw
	rec.NameFQDN = nameASCII + "." + dcn.NameASCII
	rec.NameFQDNUnicode = nameUnicode + "." + dcn.NameUnicode
	return nil
}

func setRecordNamesExtend(rec *models.RecordConfig, dcn *domaintags.DomainNameVarieties, n string) error {
	// NB(tlim): When a record has a subdomain "foo" and domain "example.com", a
	// record such as "www" is added as "www.foo" (short name) or
	// "www.foo.example.com" (FQDN name).
	// When generating the shortname, we are truncating the "D()" name, not the
	// D_EXTEND() name.  That is...  dcn.NameASCII, not
	// rec.SubDomain+dcn.NameASCII.

	nRaw := n
	nASCII := domaintags.EfficientToASCII(n)
	nUnicode := domaintags.EfficientToUnicode(nASCII)

	sdRaw := rec.SubDomain
	sdASCII := domaintags.EfficientToASCII(sdRaw)
	sdUnicode := domaintags.EfficientToUnicode(sdASCII)

	if strings.HasSuffix(nRaw, ".") {
		// The user specified a FQDN, we give it special handling.

		// with trailing dot:
		nameRawdot := nRaw
		nameASCIIdot := nASCII
		nameUnicodedot := nUnicode
		// without trailing dot:
		nameRaw := nameRawdot[:len(nameRawdot)-1]
		nameASCII := nameASCIIdot[:len(nameASCIIdot)-1]
		nameUnicode := nameUnicodedot[:len(nameUnicodedot)-1]
		// suffixes:
		dotsdsuffixASCII := "." + sdASCII + "." + dcn.NameASCII
		dotsuffixASCII := "." + dcn.NameASCII
		suffixASCII := dcn.NameASCII

		if strings.HasSuffix(nameASCII, dotsdsuffixASCII) {
			// The name is in the D_EXTEND() domain:
			rec.NameRaw = nameRaw[:len(nameRaw)-len(dcn.NameRaw)-1]
			rec.Name = nameASCII[:len(nameASCII)-len(dcn.NameASCII)-1]
			rec.NameUnicode = nameUnicode[:len(nameUnicode)-len(dcn.NameUnicode)-1]
			rec.NameFQDNRaw = nameRaw
			rec.NameFQDN = nameASCII
			rec.NameFQDNUnicode = nameUnicode
			return nil
		}
		if strings.HasSuffix(nameASCII, suffixASCII) || strings.HasSuffix(nameASCII, dotsuffixASCII) {
			// The name is NOT in the D_EXTEND() domain; but it is in the D() domain.
			return fmt.Errorf("label %q should be in the apex domain, D(%q), not D_EXTEND(%q)",
				nameRaw,
				dcn.NameRaw,
				rec.SubDomain+"."+dcn.NameRaw,
			)
		}
		// The name is not i the D_EXTEND() nor D() domain. Don't error, just return names with trailing dots.
		rec.NameRaw = nameRawdot
		rec.Name = nameASCIIdot
		rec.NameUnicode = nameUnicodedot
		rec.NameFQDNRaw = nameRawdot
		rec.NameFQDN = nameASCIIdot
		rec.NameFQDNUnicode = nameUnicodedot
		return nil
	}

	// Non-FQDN case:

	if nRaw == "@" {
		// Name is the apex. Therefore, the subdomain is the name.
		rec.NameRaw = sdRaw
		rec.Name = sdASCII
		rec.NameUnicode = sdUnicode
		rec.NameFQDNRaw = rec.NameRaw + "." + dcn.NameRaw
		rec.NameFQDN = rec.Name + "." + dcn.NameASCII
		rec.NameFQDNUnicode = rec.NameUnicode + "." + dcn.NameUnicode
		return nil
	}

	// Everything else:

	nameRaw := nRaw
	nameASCII := nASCII
	nameUnicode := nUnicode

	rec.NameRaw = nameRaw + "." + sdRaw
	rec.Name = nameASCII + "." + sdASCII
	rec.NameUnicode = nameUnicode + "." + sdUnicode
	rec.NameFQDNRaw = rec.NameRaw + "." + dcn.NameRaw
	rec.NameFQDN = rec.Name + "." + dcn.NameASCII
	rec.NameFQDNUnicode = rec.NameUnicode + "." + dcn.NameUnicode
	return nil
}

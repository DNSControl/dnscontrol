package models

import (
	"fmt"
	"strings"

	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"golang.org/x/net/idna"

	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

// ImportRawRecords iterates over the dc.RawRecords from each domain,
// converting each to a RecordConfig and deleting the raw version (to save
// memory). This is how records get from dnsconfig.js to dc.Records.
func (config *DNSConfig) ImportRawRecords() error {
	// subdomainToASCII converts a D_EXTEND() subdomain to its IDNA (punycode)
	// form. idna.ToASCII is comparatively expensive and every record in a
	// D_EXTEND block shares the same subdomain value, so the results are
	// memoized (keyed by the raw subdomain) across the whole import.
	subdomainCache := map[string]string{}
	subdomainToASCII := func(s string) (string, error) {
		if s == "" {
			return "", nil
		}
		if a, ok := subdomainCache[s]; ok {
			return a, nil
		}
		a, err := idna.ToASCII(s)
		if err != nil {
			return "", err
		}
		a = strings.ToLower(a)
		subdomainCache[s] = a
		return a, nil
	}

	for _, dc := range config.Domains {
		for _, rawRec := range dc.RawRecords {
			filePos := FixPosition(rawRec.FilePos)
			typeName := rawRec.Type

			// NB(tlim): We check if something is a builder first because LOC could be a builder or a dnsrdatav2.
			if IsBuilder(typeName) {
				records, err := dc.runBuilder(typeName, rawRec.TTL, rawRec.Args, rawRec.SubDomain)
				if err != nil {
					return err
				}
				// Annotate the .FilePos on all generated records.
				for _, record := range records {
					record.FilePos = filePos
				}
				// Generation complete!  Append the results.
				dc.Records = append(dc.Records, records...)
			} else {
				typeNum, err := dnsutilv2.StringToType(typeName)
				if err != nil {
					return fmt.Errorf("unknown record type at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
				}

				// Apply D_EXTEND() subdomain label rewriting (in Go, on
				// post-IDNA strings). Excluded types keep their label as-is.
				// The subdomain is converted to IDNA (punycode) once and reused
				// for the label, the target origin, and rec.SubDomain.
				subdomain := rawRec.SubDomain
				if subdomainExcludedTypes[typeName] {
					subdomain = ""
				} else {
					subdomain, err = subdomainToASCII(subdomain)
					if err != nil {
						return fmt.Errorf("subdomain error at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
					}
				}
				label, err := dc.LabelFromDnsconfigjs(rawRec.Args[0].(string), subdomain)
				if err != nil {
					return fmt.Errorf("label error at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
				}

				mm, err := mergeMetas(rawRec.Metas)
				if err != nil {
					return fmt.Errorf("metadata error at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
				}

				rec, err := dc.newRecordConfigFromDnsconfigjs(label, rawRec.TTL, typeNum, rawRec.Args[1:], mm, subdomain)
				if err != nil {
					return fmt.Errorf("ImportRawRecords error at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
				}
				rec.FilePos = filePos
				rec.SubDomain = subdomain
				if rec.Metadata, err = mergeMetas(rawRec.Metas); err != nil {
					return fmt.Errorf("metadata error at %s [%s(%s)]: %w", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args), err)
				}

				// The stutter check catches labels that mistakenly include the
				// zone name. It is skipped when the record opts out via
				// skip_fqdn_check, and for reverse zones, where labels are
				// legitimately written as full reverse names (e.g. via REV())
				// and are reduced to relative form later by normalization.
				if rec.Metadata["skip_fqdn_check"] != "true" && !isReverseZone(dc.Name) && doesStutter(rec.Name, dc.Name) {
					return fmt.Errorf("stutter error at %s %s(%s)", filePos, typeName, txtutil.ZoneifyManyAny(rawRec.Args))
				}

				// Conversion complete!  Append it.
				dc.AddRecordConfig(rec)
			}

			// We're never going to see this rawRec again. Free its Args.
			clear(rawRec.Args)
			rawRec.Args = nil
		}

		// We're never going to see these RawRecords again. Let them go.
		dc.RawRecords = nil
	}

	return nil
}

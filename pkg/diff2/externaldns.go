package diff2

// This file implements the IGNORE_EXTERNAL_DNS feature that automatically
// detects and ignores DNS records managed by Kubernetes external-dns.
//
// External-dns uses TXT records to track ownership of DNS records it manages.
// The TXT record format is:
//   "heritage=external-dns,external-dns/owner=<owner-id>,external-dns/resource=<resource>"
//
// External-dns TXT record naming conventions:
// - For A records: prefix + original name (e.g., "a-myapp.example.com" for "myapp.example.com")
// - For CNAME records: prefix + original name (e.g., "cname-myapp.example.com")
// - Default prefixes: "a-", "aaaa-", "cname-", "ns-", "mx-"
// - Can also use --txt-prefix or --txt-suffix flags in external-dns

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
)

const (
	// externalDNSHeritage is the heritage value that external-dns uses in its TXT records.
	externalDNSHeritage = "heritage=external-dns"
)

// externalDNSManagedRecord represents a record managed by external-dns.
type externalDNSManagedRecord struct {
	Label      string // The label of the managed record (without domain suffix)
	RecordType string // The type of the managed record (A, AAAA, CNAME, etc.)
}

// isExternalDNSTxtRecord checks if a TXT record is an external-dns ownership record.
// It returns true and the managed record info if it is, false otherwise.
// customPrefix is an optional prefix that external-dns was configured with (e.g., "extdns-").
func isExternalDNSTxtRecord(rec *models.RecordConfig, domain string, customPrefix string) (bool, *externalDNSManagedRecord) {
	if rec.Type != "TXT" {
		return false, nil
	}

	// Get the TXT record content
	target := rec.GetTargetTXTJoined()

	// Check if it contains the external-dns heritage marker
	if !strings.Contains(target, externalDNSHeritage) {
		return false, nil
	}

	// This is an external-dns TXT record. Now we need to figure out what record it manages.
	// External-dns TXT record naming:
	// - New format with record type prefix: "a-myapp.example.com" manages "myapp.example.com" A record
	// - Old format without type: "myapp.example.com" (legacy, manages the record with same name)
	// - With custom prefix: e.g., "externaldns-a-myapp.example.com"
	// - With custom suffix: e.g., "myapp-externaldns.example.com"

	label := rec.GetLabel()
	managed := parseExternalDNSTxtLabel(label, customPrefix)

	return true, managed
}

// parseExternalDNSTxtLabel parses an external-dns TXT record label to extract
// the managed record information.
//
// External-dns uses these prefixes by default (when using %{record_type} in prefix):
// - "a-" for A records
// - "aaaa-" for AAAA records
// - "cname-" for CNAME records
// - "ns-" for NS records
// - "mx-" for MX records
//
// Without %{record_type}, it just uses the prefix directly, and the record type
// is encoded as "a-", "cname-", etc. at the start of the label.
//
// If customPrefix is non-empty, it will be stripped first before looking for
// record type prefixes.
func parseExternalDNSTxtLabel(label string, customPrefix string) *externalDNSManagedRecord {
	workingLabel := label

	// If a custom prefix is specified, strip it first
	if customPrefix != "" {
		if strings.HasPrefix(strings.ToLower(workingLabel), strings.ToLower(customPrefix)) {
			workingLabel = workingLabel[len(customPrefix):]
		}
		// else: Custom prefix specified but not found - this might be a legacy record
		// Continue with original label
	}

	// Standard prefixes used by external-dns
	// Supports both hyphen format (a-www) and period format (a.www)
	// Period format is used when --txt-prefix includes %{record_type}.
	prefixes := []struct {
		prefix     string
		recordType string
	}{
		{"aaaa.", "AAAA"}, // Period format - must check before "a."
		{"aaaa-", "AAAA"}, // Hyphen format - must check before "a-"
		{"a.", "A"},       // Period format
		{"a-", "A"},       // Hyphen format
		{"cname.", "CNAME"},
		{"cname-", "CNAME"},
		{"ns.", "NS"},
		{"ns-", "NS"},
		{"mx.", "MX"},
		{"mx-", "MX"},
		{"srv.", "SRV"},
		{"srv-", "SRV"},
		{"txt.", "TXT"},
		{"txt-", "TXT"},
	}

	for _, p := range prefixes {
		if strings.HasPrefix(strings.ToLower(workingLabel), p.prefix) {
			managedLabel := workingLabel[len(p.prefix):]
			// managedLabel is already lowercase from the prefix match
			// Handle the case where the managed label is empty (apex domain)
			if managedLabel == "" {
				managedLabel = "@"
			}
			return &externalDNSManagedRecord{
				Label:      managedLabel,
				RecordType: p.recordType,
			}
		}
	}

	// If custom prefix was specified and stripped, check if the remaining label
	// is a record type indicator (for period format apex domains: extdns-a. at apex becomes extdns-a)
	if customPrefix != "" && workingLabel != label {
		// Check if remaining label is just a record type (apex domain with period format)
		// e.g., prefix "extdns-" with label "extdns-a" → workingLabel "a" → apex A record
		apexRecordTypes := map[string]string{
			"a":     "A",
			"aaaa":  "AAAA",
			"cname": "CNAME",
			"ns":    "NS",
			"mx":    "MX",
			"srv":   "SRV",
			"txt":   "TXT",
		}
		if recType, ok := apexRecordTypes[strings.ToLower(workingLabel)]; ok {
			return &externalDNSManagedRecord{
				Label:      "@",
				RecordType: recType,
			}
		}

		// The prefix was stripped but no record type found
		// This means it's a simple prefix like "extdns-" without record type
		// We can't determine the record type, so match all types
		if workingLabel == "" {
			workingLabel = "@"
		}
		return &externalDNSManagedRecord{
			Label:      workingLabel,
			RecordType: "", // Empty means match any type
		}
	}

	// No recognized prefix - this might be a legacy format or custom prefix
	// In legacy format, the TXT record has the same name as the managed record
	// We can't determine the record type in this case, so we'll match all types
	return &externalDNSManagedRecord{
		Label:      label,
		RecordType: "", // Empty means match any type
	}
}

// findExternalDNSManagedRecords scans the existing records for external-dns TXT records
// and builds a map of records that are managed by external-dns.
// Returns a map keyed by "label:type" -> true for managed records
// customPrefix is an optional prefix that external-dns was configured with (e.g., "extdns-").
func findExternalDNSManagedRecords(existing models.Records, domain string, customPrefix string) map[string]bool {
	managed := make(map[string]bool)

	// Scan all external-dns TXT records
	for _, rec := range existing {
		isExtDNS, info := isExternalDNSTxtRecord(rec, domain, customPrefix)
		if isExtDNS && info != nil {
			// Mark the TXT record itself as managed
			txtKey := rec.GetLabel() + ":TXT"
			managed[txtKey] = true

			// Mark the record that this TXT record manages
			if info.RecordType != "" {
				// Specific record type
				key := info.Label + ":" + info.RecordType
				managed[key] = true
			} else {
				// Legacy format - we need to find matching records
				// We'll mark this label as managed for common record types
				for _, rtype := range []string{"A", "AAAA", "CNAME", "NS", "MX", "SRV"} {
					key := info.Label + ":" + rtype
					managed[key] = true
				}
			}
		}
	}

	return managed
}

// filterExternalDNSRecords takes a list of existing records and returns those
// that should be ignored because they are managed by external-dns.
// customPrefix is an optional prefix that external-dns was configured with (e.g., "extdns-").
func filterExternalDNSRecords(existing models.Records, domain string, customPrefix string) models.Records {
	managedMap := findExternalDNSManagedRecords(existing, domain, customPrefix)
	if len(managedMap) == 0 {
		return nil
	}

	var ignored models.Records
	for _, rec := range existing {
		key := rec.GetLabel() + ":" + rec.Type
		if managedMap[key] {
			ignored = append(ignored, rec)
		}
	}

	return ignored
}

// GetExternalDNSIgnoredRecords returns the records that should be ignored
// because they are managed by external-dns. This is called from handsoff()
// when IgnoreExternalDNS is enabled for a domain.
// customPrefix is an optional prefix that external-dns was configured with (e.g., "extdns-").
func GetExternalDNSIgnoredRecords(existing models.Records, domain string, customPrefix string) models.Records {
	return filterExternalDNSRecords(existing, domain, customPrefix)
}

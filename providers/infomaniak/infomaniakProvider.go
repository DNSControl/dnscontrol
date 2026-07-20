package infomaniak

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/miekg/dns/dnsutil"
)

// infomaniakProvider is the handle for operations.
type infomaniakProvider struct {
	apiToken string // the account access token
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanUseCAA:   providers.Can(),
	providers.CanUseDNAME: providers.Can(),
	providers.CanUseDS:    providers.Can(),
	providers.CanUseSSHFP: providers.Can(),
	providers.CanUseTLSA:  providers.Can(),
	providers.CanUseSRV:   providers.Can(),
	// providers.DocCreateDomains: providers.Can(),
}

func newInfomaniak(m map[string]string, message json.RawMessage) (providers.DNSServiceProvider, error) {
	api := &infomaniakProvider{}
	api.apiToken = m["token"]
	if api.apiToken == "" {
		return nil, errors.New("missing Infomaniak personal access token")
	}

	return api, nil
}

func init() {
	const providerName = "INFOMANIAK"
	const providerMaintainer = "@jbelien"
	fns := providers.DspFuncs{
		Initializer:   newInfomaniak,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

func (p *infomaniakProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	zone, err := p.getDNSZone(domain)
	if err != nil {
		return nil, err
	}

	return models.ToNameservers(zone.Nameservers)
}

// addTrailingDot adds a trailing dot if it's missing.
func addTrailingDot(target string) string {
	if target == "" || target == "." {
		return target
	}
	if !strings.HasSuffix(target, ".") {
		return target + "."
	}
	return target
}

// toRecordConfig converts a DNS record from Infomaniak API to RecordConfig.
func toRecordConfig(domain string, r dnsRecord) (*models.RecordConfig, error) {
	rc := &models.RecordConfig{
		TTL:      uint32(r.TTL),
		Original: r,
	}

	// Handle the source/label - Infomaniak uses empty string or "." for apex
	label := r.Source
	if label == "" || label == "." {
		label = "@"
	}
	rc.SetLabel(label, domain)

	// Parse the target based on record type
	rtype := r.Type
	target := r.Target

	var err error
	switch rtype {
	case "A", "AAAA":
		rc.Type = rtype
		err = rc.SetTarget(target)

	case "CNAME", "NS", "DNAME":
		rc.Type = rtype
		// Add trailing dot and use AddOrigin to properly qualify the target
		err = rc.SetTarget(dnsutil.AddOrigin(addTrailingDot(target), domain))

	case "MX":
		// Infomaniak returns MX as "priority target" (e.g., "5 mta-gw.infomaniak.ch")
		rc.Type = rtype
		err = rc.SetTargetMXString(addTrailingDot(target))

	case "TXT":
		rc.Type = rtype
		// Infomaniak API returns TXT values wrapped in quotes, strip them
		if len(target) >= 2 && strings.HasPrefix(target, "\"") && strings.HasSuffix(target, "\"") {
			target = target[1 : len(target)-1]
		}
		err = rc.SetTargetTXT(target)

	case "SRV":
		// Infomaniak returns SRV as "priority weight port target"
		rc.Type = rtype
		err = rc.SetTargetSRVString(addTrailingDot(target))

	case "CAA":
		// Infomaniak returns CAA as "flags tag value" (e.g., "0 issue letsencrypt.org")
		rc.Type = rtype
		err = rc.SetTargetCAAString(target)

	case "DS":
		// Infomaniak returns DS as "keytag algorithm digesttype digest"
		// Note: Infomaniak may split long digest data with spaces, so we need to rejoin them
		rc.Type = rtype
		parts := strings.Fields(target)
		if len(parts) >= 4 {
			// Rejoin all parts after the first 3 (keytag, algorithm, digesttype) as the digest
			digest := strings.Join(parts[3:], "")
			target = fmt.Sprintf("%s %s %s %s", parts[0], parts[1], parts[2], digest)
		}
		err = rc.SetTargetDSString(target)

	case "SSHFP":
		// Infomaniak returns SSHFP as "algorithm fingerprint_type fingerprint"
		// Note: Infomaniak may split long fingerprint data with spaces, so we need to rejoin them
		rc.Type = rtype
		parts := strings.Fields(target)
		if len(parts) >= 3 {
			// Rejoin all parts after the first 2 (algorithm, fingerprint_type) as the fingerprint
			fingerprint := strings.Join(parts[2:], "")
			target = fmt.Sprintf("%s %s %s", parts[0], parts[1], fingerprint)
		}
		err = rc.SetTargetSSHFPString(target)

	case "TLSA":
		// Infomaniak returns TLSA as "usage selector matching_type certificate"
		// Note: Infomaniak may split long certificate data with spaces, so we need to rejoin them
		rc.Type = rtype
		parts := strings.Fields(target)
		if len(parts) >= 4 {
			// Rejoin all parts after the first 3 (usage, selector, matching_type) as the certificate
			certificate := strings.Join(parts[3:], "")
			target = fmt.Sprintf("%s %s %s %s", parts[0], parts[1], parts[2], certificate)
		}
		err = rc.SetTargetTLSAString(target)

	default:
		rc.Type = rtype
		err = rc.SetTarget(target)
	}

	if err != nil {
		return nil, fmt.Errorf("unparsable record type=%q target=%q received from Infomaniak: %w", rtype, target, err)
	}

	return rc, nil
}

// fromRecordConfig converts a RecordConfig to the API format for creation.
func fromRecordConfig(rc *models.RecordConfig) *dnsRecordCreate {
	// Get the label - Infomaniak uses empty string for apex
	label := rc.GetLabel()
	if label == "@" {
		label = ""
	}

	// Get the target in the format expected by Infomaniak API
	var target string
	switch rc.Type {
	case "A", "AAAA":
		target = rc.GetTargetField()

	case "CNAME", "NS", "DNAME":
		// Remove trailing dot for the API
		target = strings.TrimSuffix(rc.GetTargetField(), ".")

	case "MX":
		// Format: "priority target" (without trailing dot)
		target = fmt.Sprintf("%d %s", rc.MxPreference, strings.TrimSuffix(rc.GetTargetField(), "."))

	case "TXT":
		target = rc.GetTargetField()

	case "SRV":
		// Format: "priority weight port target" (without trailing dot)
		target = fmt.Sprintf("%d %d %d %s", rc.SrvPriority, rc.SrvWeight, rc.SrvPort, strings.TrimSuffix(rc.GetTargetField(), "."))

	case "CAA":
		// Format: "flags tag value"
		target = fmt.Sprintf("%d %s %s", rc.CaaFlag, rc.CaaTag, rc.GetTargetField())

	case "DS":
		// Format: "keytag algorithm digesttype digest"
		target = fmt.Sprintf("%d %d %d %s", rc.DsKeyTag, rc.DsAlgorithm, rc.DsDigestType, rc.DsDigest)

	case "SSHFP":
		// Format: "algorithm fingerprint_type fingerprint"
		target = fmt.Sprintf("%d %d %s", rc.SshfpAlgorithm, rc.SshfpFingerprint, rc.GetTargetField())

	case "TLSA":
		// Format: "usage selector matching_type certificate"
		target = fmt.Sprintf("%d %d %d %s", rc.TlsaUsage, rc.TlsaSelector, rc.TlsaMatchingType, rc.GetTargetField())

	default:
		target = rc.GetTargetField()
	}

	return &dnsRecordCreate{
		Source: label,
		Type:   rc.Type,
		TTL:    int64(rc.TTL),
		Target: target,
	}
}

// toRecordUpdate converts a RecordConfig to the API format for updating.
func toRecordUpdate(rc *models.RecordConfig) *dnsRecordUpdate {
	// Get the target in the format expected by Infomaniak API
	var target string
	switch rc.Type {
	case "A", "AAAA":
		target = rc.GetTargetField()

	case "CNAME", "NS", "DNAME":
		// Remove trailing dot for the API
		target = strings.TrimSuffix(rc.GetTargetField(), ".")

	case "MX":
		// Format: "priority target" (without trailing dot)
		target = fmt.Sprintf("%d %s", rc.MxPreference, strings.TrimSuffix(rc.GetTargetField(), "."))

	case "TXT":
		target = rc.GetTargetField()

	case "SRV":
		// Format: "priority weight port target" (without trailing dot)
		target = fmt.Sprintf("%d %d %d %s", rc.SrvPriority, rc.SrvWeight, rc.SrvPort, strings.TrimSuffix(rc.GetTargetField(), "."))

	case "CAA":
		// Format: "flags tag value"
		target = fmt.Sprintf("%d %s %s", rc.CaaFlag, rc.CaaTag, rc.GetTargetField())

	case "DS":
		// Format: "keytag algorithm digesttype digest"
		target = fmt.Sprintf("%d %d %d %s", rc.DsKeyTag, rc.DsAlgorithm, rc.DsDigestType, rc.DsDigest)

	case "SSHFP":
		// Format: "algorithm fingerprint_type fingerprint"
		target = fmt.Sprintf("%d %d %s", rc.SshfpAlgorithm, rc.SshfpFingerprint, rc.GetTargetField())

	case "TLSA":
		// Format: "usage selector matching_type certificate"
		target = fmt.Sprintf("%d %d %d %s", rc.TlsaUsage, rc.TlsaSelector, rc.TlsaMatchingType, rc.GetTargetField())

	default:
		target = rc.GetTargetField()
	}

	return &dnsRecordUpdate{
		TTL:    int64(rc.TTL),
		Target: target,
	}
}

func (p *infomaniakProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := p.getDNSRecords(domain)
	if err != nil {
		return nil, err
	}

	cleanRecords := make(models.Records, 0, len(records))

	for _, r := range records {
		recConfig, err := toRecordConfig(domain, r)
		if err != nil {
			return nil, err
		}
		cleanRecords = append(cleanRecords, recConfig)
	}

	return cleanRecords, nil
}

func (p *infomaniakProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction
	domain := dc.Name

	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CHANGE:
			oldRec := change.Old[0].Original.(dnsRecord)
			newRec := change.New[0]
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F: func() error {
					_, err := p.updateDNSRecord(domain, fmt.Sprintf("%v", oldRec.ID), toRecordUpdate(newRec))
					return err
				},
			})

		case diff2.CREATE:
			rec := change.New[0]
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F: func() error {
					_, err := p.createDNSRecord(domain, fromRecordConfig(rec))
					return err
				},
			})

		case diff2.DELETE:
			rec := change.Old[0].Original.(dnsRecord)
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F: func() error {
					return p.deleteDNSRecord(domain, fmt.Sprintf("%v", rec.ID))
				},
			})
		}
	}

	return corrections, actualChangeCount, nil
}

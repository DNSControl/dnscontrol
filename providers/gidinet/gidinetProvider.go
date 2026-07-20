package gidinet

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/miekg/dns/dnsutil"
)

/*
Gidinet DNS API provider:

Info required in `creds.json`:
   - username
   - password

Note on Registrar functionality:
   The registrar API (for managing nameservers via NAMESERVER()) requires
   API reseller account credentials. Regular customer API credentials can
   only manage DNS records, not domain registration settings.

   TODO: Test with customer API credentials to document the specific error
   returned when attempting to set NS records, and provide a clearer message.
*/

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanConcur:              providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseCAA:              providers.Cannot("Only premium service"),
	providers.CanUseDHCID:            providers.Cannot(),
	providers.CanUseDNAME:            providers.Cannot(),
	providers.CanUseDNSKEY:           providers.Cannot(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseHTTPS:            providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseSVCB:             providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Cannot("Must be created via web UI"),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "GIDINET"
	const providerMaintainer = "@zupolgec"

	// Register as DNS provider
	fns := providers.DspFuncs{
		Initializer:   NewGidinet,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)

	// Register as Registrar
	providers.RegisterRegistrarType(providerName, newRegistrar)

	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Gidinet",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/gidinet",
		PortalURL:   "https://www.gidinet.com/modules/private/account_password/",
		Notes:       "The username is your usual Gidinet account username, but the password must be the dedicated \"API password\" generated in the Gidinet control panel — not your normal login password.",
		Fields: []providers.CredsField{
			{
				Key:      "username",
				Label:    "Username",
				Help:     "Your Gidinet account username (same as the control-panel login).",
				Required: true,
			},
			{
				Key:      "password",
				Label:    "API password",
				Help:     "The \"API password\" generated at https://www.gidinet.com/modules/private/account_password/ — not your normal login password.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

// newRegistrar creates a new Gidinet registrar instance.
func newRegistrar(m map[string]string) (providers.Registrar, error) {
	if m["username"] == "" {
		return nil, errors.New("missing Gidinet username")
	}
	if m["password"] == "" {
		return nil, errors.New("missing Gidinet password")
	}
	return newClient(m["username"], m["password"]), nil
}

// NewGidinet creates a new Gidinet DNS provider.
func NewGidinet(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	if m["username"] == "" {
		return nil, errors.New("missing Gidinet username")
	}
	if m["password"] == "" {
		return nil, errors.New("missing Gidinet password")
	}

	api := newClient(m["username"], m["password"])
	return api, nil
}

// GetNameservers returns the static Gidinet DNS nameservers used by every
// zone hosted on the platform. Returning them here lets dnscontrol suggest
// the correct delegation to the registrar without requiring an explicit
// NAMESERVER() in the zone file.
func (c *gidinetProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers([]string{
		"dnsl1.gidinet.com",
		"dnsl2.gidinet.com",
	})
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *gidinetProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := c.recordGetList(domain)
	if err != nil {
		return nil, err
	}

	var existingRecords []*models.RecordConfig
	for _, r := range records {
		// Skip read-only records (usually NS records at apex managed by registrar)
		if r.ReadOnly {
			continue
		}
		// Skip suspended records
		if r.Suspended {
			continue
		}

		rc, err := toRecordConfig(domain, r)
		if err != nil {
			return nil, err
		}
		existingRecords = append(existingRecords, rc)
	}

	return existingRecords, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (c *gidinetProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction

	// Filter out apex NS records - they are managed by the registrar, not the DNS provider
	filterApexNS(dc)

	// Normalize TTL values to allowed Gidinet values
	for _, record := range dc.Records {
		record.TTL = fixTTL(record.TTL)
	}

	instructions, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	for _, inst := range instructions {
		switch inst.Type {
		case diff2.REPORT:
			corrections = append(corrections,
				&models.Correction{
					Msg: inst.MsgsJoined,
				})
			continue

		case diff2.CREATE:
			rec := inst.New[0]
			dnsRec := toGidinetRecord(dc.Name, rec)
			corrections = append(corrections, &models.Correction{
				Msg: inst.MsgsJoined,
				F: func() error {
					return c.recordAdd(dnsRec)
				},
			})

		case diff2.CHANGE:
			oldRec := inst.Old[0]
			newRec := inst.New[0]
			oldDNS := oldRec.Original.(*DNSRecordListItem)
			oldGidinet := &DNSRecord{
				DomainName: oldDNS.DomainName,
				HostName:   oldDNS.HostName,
				RecordType: oldDNS.RecordType,
				Data:       oldDNS.Data,
				TTL:        oldDNS.TTL,
				Priority:   oldDNS.Priority,
			}
			newGidinet := toGidinetRecord(dc.Name, newRec)
			corrections = append(corrections, &models.Correction{
				Msg: inst.MsgsJoined,
				F: func() error {
					return c.recordUpdate(oldGidinet, newGidinet)
				},
			})

		case diff2.DELETE:
			oldRec := inst.Old[0]
			oldDNS := oldRec.Original.(*DNSRecordListItem)
			dnsRec := &DNSRecord{
				DomainName: oldDNS.DomainName,
				HostName:   oldDNS.HostName,
				RecordType: oldDNS.RecordType,
				Data:       oldDNS.Data,
				TTL:        oldDNS.TTL,
				Priority:   oldDNS.Priority,
			}
			corrections = append(corrections, &models.Correction{
				Msg: inst.MsgsJoined,
				F: func() error {
					return c.recordDelete(dnsRec)
				},
			})

		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

// toRecordConfig converts a Gidinet DNS record to a RecordConfig.
func toRecordConfig(domain string, r *DNSRecordListItem) (*models.RecordConfig, error) {
	rc := &models.RecordConfig{
		Type:     r.RecordType,
		TTL:      uint32(r.TTL),
		Original: r,
	}

	// Set the label from the hostname
	// Gidinet returns full hostnames like "www.domain.com"
	label := fromFQDN(r.HostName, domain)
	rc.SetLabel(label, domain)

	// Handle different record types
	switch r.RecordType {
	case "MX":
		rc.MxPreference = uint16(r.Priority)
		// MX target should be FQDN with trailing dot
		target := r.Data
		if !strings.HasSuffix(target, ".") {
			target = dnsutil.AddOrigin(target+".", domain)
		}
		if err := rc.SetTarget(target); err != nil {
			return nil, err
		}

	case "SRV":
		// SRV records in Gidinet: Data contains "priority weight port target"
		// The Priority field is only for MX records, not SRV
		parts := strings.Fields(r.Data)
		if len(parts) >= 4 {
			priority, _ := strconv.ParseUint(parts[0], 10, 16)
			weight, _ := strconv.ParseUint(parts[1], 10, 16)
			port, _ := strconv.ParseUint(parts[2], 10, 16)
			target := parts[3]
			if !strings.HasSuffix(target, ".") {
				target = target + "."
			}
			rc.SrvPriority = uint16(priority)
			rc.SrvWeight = uint16(weight)
			rc.SrvPort = uint16(port)
			if err := rc.SetTarget(target); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid SRV data format: %q (expected: priority weight port target)", r.Data)
		}

	case "CNAME", "NS":
		target := r.Data
		if !strings.HasSuffix(target, ".") {
			target = dnsutil.AddOrigin(target+".", domain)
		}
		if err := rc.SetTarget(target); err != nil {
			return nil, err
		}

	case "TXT":
		// Gidinet may return TXT values in chunked format: "chunk1" "chunk2"
		// Use unchunkTXT to parse back to a single string
		txtData := unchunkTXT(r.Data)
		if err := rc.SetTargetTXT(txtData); err != nil {
			return nil, err
		}

	default: // A, AAAA, etc.
		if err := rc.SetTarget(r.Data); err != nil {
			return nil, err
		}
	}

	return rc, nil
}

// toGidinetRecord converts a RecordConfig to a Gidinet DNS record.
func toGidinetRecord(domain string, rc *models.RecordConfig) *DNSRecord {
	rec := &DNSRecord{
		DomainName: domain,
		HostName:   toFQDN(rc.GetLabel(), domain),
		RecordType: rc.Type,
		TTL:        int(rc.TTL),
		Priority:   0,
	}

	switch rc.Type {
	case "MX":
		rec.Priority = int(rc.MxPreference)
		// Remove trailing dot from target
		target := rc.GetTargetField()
		rec.Data = strings.TrimSuffix(target, ".")

	case "SRV":
		// SRV Data format: priority weight port target (with trailing dot per Gidinet spec)
		// The Priority field is only for MX records
		rec.Priority = 0
		target := rc.GetTargetField()
		if !strings.HasSuffix(target, ".") {
			target = target + "."
		}
		rec.Data = fmt.Sprintf("%d %d %d %s", rc.SrvPriority, rc.SrvWeight, rc.SrvPort, target)

	case "CNAME", "NS":
		// Remove trailing dot from target
		target := rc.GetTargetField()
		rec.Data = strings.TrimSuffix(target, ".")

	case "TXT":
		// Chunk long TXT values into quoted segments for the API
		rec.Data = chunkTXT(rc.GetTargetTXTJoined())

	default: // A, AAAA, etc.
		rec.Data = rc.GetTargetField()
	}

	return rec
}

// filterApexNS removes NS records at the apex from dc.Records.
// Gidinet does not support modifying apex NS records via the DNS API - they are
// managed by the registrar. Apex NS records synthesized from dc.Nameservers
// (auto-injected by GetNameservers or declared via NAMESERVER()) are dropped
// silently since they are handled by the registrar side. Any other apex NS
// record is dropped with a warning because it cannot be honored by the
// Gidinet DNS API.
func filterApexNS(dc *models.DomainConfig) {
	expected := make(map[string]bool, len(dc.Nameservers))
	for _, ns := range dc.Nameservers {
		expected[strings.TrimSuffix(ns.Name, ".")] = true
	}

	newList := make([]*models.RecordConfig, 0, len(dc.Records))
	for _, rec := range dc.Records {
		if rec.Type == "NS" && rec.GetLabelFQDN() == dc.Name {
			target := strings.TrimSuffix(rec.GetTargetField(), ".")
			if !expected[target] {
				printer.Warnf("GIDINET does not support modifying NS records at apex. %s will not be added.\n", rec.GetTargetField())
			}
			continue
		}
		newList = append(newList, rec)
	}
	dc.Records = newList
}

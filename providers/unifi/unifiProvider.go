package unifi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

// Provider metadata.
const (
	providerName       = "UNIFI"
	providerMaintainer = "@zupolgec"
)

// Provider capabilities.
var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	providers.CanGetZones:            providers.Cannot("UniFi stores records flat, not by zone"),
	providers.CanConcur:              providers.Cannot(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseCAA:              providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
	providers.DocCreateDomains:       providers.Cannot("UniFi does not have zone concept"),
}

// unifiProvider implements the DNSServiceProvider interface for UniFi Network.
type unifiProvider struct {
	client *unifiClient
}

func init() {
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

func newDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newUnifi(conf, metadata)
}

// newUnifi creates a new UniFi provider from configuration.
func newUnifi(m map[string]string, _ json.RawMessage) (*unifiProvider, error) {
	host := m["host"]
	consoleID := m["console_id"]
	apiKey := m["api_key"]
	site := m["site"]
	skipTLS := strings.EqualFold(m["skip_tls_verify"], "true")
	debug := strings.EqualFold(m["debug"], "true")

	// API version: "auto" (default), "new", or "legacy"
	apiVersion := strings.ToLower(m["api_version"])
	if apiVersion == "" {
		apiVersion = "auto"
	}
	if apiVersion != "auto" && apiVersion != "new" && apiVersion != "legacy" {
		return nil, fmt.Errorf("invalid api_version '%s': must be 'auto', 'new', or 'legacy'", apiVersion)
	}

	// Validate required fields
	if apiKey == "" {
		return nil, errors.New("missing UniFi api_key")
	}
	if site == "" {
		site = "default"
	}

	// Must have either host (local) or console_id (cloud)
	if host == "" && consoleID == "" {
		return nil, errors.New("missing UniFi host or console_id")
	}

	client := newClient(host, consoleID, apiKey, site, apiVersion, skipTLS, debug)

	return &unifiProvider{
		client: client,
	}, nil
}

// GetNameservers returns the nameservers for a domain.
// UniFi is used for local DNS, so we return an empty list.
func (p *unifiProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return []*models.Nameserver{}, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (p *unifiProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	// Fetch all records from UniFi using the appropriate API
	allRecords, isNewAPI, err := p.client.getRecords()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch records from UniFi: %w", err)
	}

	// Filter records that belong to this domain
	var records models.Records
	domainSuffix := "." + domain

	for _, r := range allRecords {
		var rc *models.RecordConfig
		var fqdn string

		if isNewAPI {
			newRec := r.(*dnsPolicyRecord)
			fqdn = newRec.Domain

			// Check if this record belongs to our domain
			if fqdn != domain && !strings.HasSuffix(fqdn, domainSuffix) {
				continue
			}

			rc, err = newToRecord(domain, newRec)
			if err != nil {
				return nil, fmt.Errorf("failed to convert record %s: %w", fqdn, err)
			}
		} else {
			legacyRec := r.(*legacyDNSRecord)
			fqdn = legacyRec.Key

			// Check if this record belongs to our domain
			if fqdn != domain && !strings.HasSuffix(fqdn, domainSuffix) {
				continue
			}

			rc, err = legacyToRecord(domain, legacyRec)
			if err != nil {
				return nil, fmt.Errorf("failed to convert record %s: %w", fqdn, err)
			}
		}

		records = append(records, rc)
	}

	return records, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (p *unifiProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction

	// UniFi doesn't care about TTL much, but we normalize it
	for _, record := range dc.Records {
		if record.TTL == 0 {
			record.TTL = 300
		}
	}

	// Use diff2.ByRecord since each record has a unique ID
	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	// Determine which API to use
	useNewAPI := p.client.useNewAPI()

	for _, change := range changes {
		var corr *models.Correction

		switch change.Type {
		case diff2.REPORT:
			corr = &models.Correction{Msg: change.MsgsJoined}

		case diff2.CREATE:
			newRec := change.New[0]
			if useNewAPI {
				newAPIRec, err := recordToNew(newRec)
				if err != nil {
					return nil, 0, fmt.Errorf("failed to convert record for create: %w", err)
				}
				corr = &models.Correction{
					Msg: change.Msgs[0],
					F: func() error {
						_, err := p.client.createRecordNew(newAPIRec)
						return err
					},
				}
			} else {
				legacyMap, err := recordToLegacyMap(newRec)
				if err != nil {
					return nil, 0, fmt.Errorf("failed to convert record for create: %w", err)
				}
				corr = &models.Correction{
					Msg: change.Msgs[0],
					F: func() error {
						_, err := p.client.createRecordLegacy(legacyMap)
						return err
					},
				}
			}

		case diff2.CHANGE:
			oldRec := change.Old[0]
			newRec := change.New[0]
			id := getRecordID(oldRec)
			if id == "" {
				return nil, 0, fmt.Errorf("cannot update record without ID: %s", oldRec.NameFQDN)
			}
			if useNewAPI {
				newAPIRec, err := recordToNew(newRec)
				if err != nil {
					return nil, 0, fmt.Errorf("failed to convert record for update: %w", err)
				}
				corr = &models.Correction{
					Msg: fmt.Sprintf("%s (unifi id: %s)", change.Msgs[0], id),
					F: func() error {
						_, err := p.client.updateRecordNew(id, newAPIRec)
						return err
					},
				}
			} else {
				legacyMap, err := recordToLegacyMap(newRec)
				if err != nil {
					return nil, 0, fmt.Errorf("failed to convert record for update: %w", err)
				}
				corr = &models.Correction{
					Msg: fmt.Sprintf("%s (unifi id: %s)", change.Msgs[0], id),
					F: func() error {
						_, err := p.client.updateRecordLegacy(id, legacyMap)
						return err
					},
				}
			}

		case diff2.DELETE:
			oldRec := change.Old[0]
			id := getRecordID(oldRec)
			if id == "" {
				return nil, 0, fmt.Errorf("cannot delete record without ID: %s", oldRec.NameFQDN)
			}
			if useNewAPI {
				corr = &models.Correction{
					Msg: fmt.Sprintf("%s (unifi id: %s)", change.Msgs[0], id),
					F: func() error {
						return p.client.deleteRecordNew(id)
					},
				}
			} else {
				corr = &models.Correction{
					Msg: fmt.Sprintf("%s (unifi id: %s)", change.Msgs[0], id),
					F: func() error {
						return p.client.deleteRecordLegacy(id)
					},
				}
			}

		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}

		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}

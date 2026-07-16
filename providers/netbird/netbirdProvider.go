package netbird

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

// supportedRecordTypes is the set of DNS record types supported by NetBird.
var supportedRecordTypes = map[string]bool{
	"A":     true,
	"AAAA":  true,
	"CNAME": true,
}

/*

NetBird API DNS provider:

Info required in `creds.json`:
   - token

API documentation: https://docs.netbird.io/api/resources/dns-zones

*/

const (
	netbirdAPIURL = "https://api.netbird.io/api"
)

// netbirdProvider is the handle for operations.
type netbirdProvider struct {
	token   string
	client  *http.Client
	apiURL  string
	zoneMu  sync.Mutex           // Protects zoneMap
	zoneMap map[string]*zoneInfo // Cache of zone info by domain
}

// NewNetbird creates a NetBird-specific DNS provider.
func NewNetbird(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	if m["token"] == "" {
		return nil, errors.New("no NetBird token provided")
	}

	apiUrl := m["apiurl"]
	if apiUrl == "" {
		apiUrl = netbirdAPIURL
	}

	api := &netbirdProvider{
		token:   m["token"],
		client:  &http.Client{},
		apiURL:  apiUrl,
		zoneMap: make(map[string]*zoneInfo),
	}

	// Test the token by listing zones
	_, err := api.listZones()
	if err != nil {
		return nil, fmt.Errorf("NetBird token validation failed: %w", err)
	}

	return api, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanConcur:              providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseCAA:              providers.Cannot(),
	providers.CanUseDHCID:            providers.Cannot(),
	providers.CanUseDNAME:            providers.Cannot(),
	providers.CanUseDNSKEY:           providers.Cannot(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseHTTPS:            providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Cannot(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseSMIMEA:           providers.Cannot(),
	providers.CanUseSVCB:             providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "NETBIRD"
	const providerMaintainer = "@yzqzss"
	fns := providers.DspFuncs{
		Initializer:   NewNetbird,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Netbird",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/netbird",
		PortalURL:   "https://app.netbird.io/settings",
		Fields: []providers.CredsField{
			{Key: "token", Label: "API Token", Required: true, Secret: true},
			{Key: "apiurl", Label: "API Url", Default: netbirdAPIURL},
		},
	})
}

// AuditRecords returns a list of errors for records that aren't supported.
func AuditRecords(records []*models.RecordConfig) []error {
	var errs []error
	for _, rc := range records {
		if !supportedRecordTypes[rc.Type] {
			errs = append(errs, fmt.Errorf("NETBIRD does not support %s records", rc.Type))
		}
	}
	return errs
}

// parseEnabled parses the "netbird_enabled" metadata field.
// Returns nil if not set (don't change), &true or &false if explicitly set.
func parseEnabled(metadata map[string]string) *bool {
	if v, ok := metadata["netbird_enabled"]; ok {
		result := v != "false"
		return &result
	}
	return nil
}

// parseEnableSearchDomain parses the "netbird_enable_search_domain" metadata field.
// Returns nil if not set (don't change), &true or &false if explicitly set.
func parseEnableSearchDomain(metadata map[string]string) *bool {
	if v, ok := metadata["netbird_enable_search_domain"]; ok {
		result := v == "true"
		return &result
	}
	return nil
}

// EnsureZoneExists creates a zone if it does not exist, or updates it if metadata specifies different settings.
func (api *netbirdProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	metadata := dc.Metadata

	zones, err := api.listZones()
	if err != nil {
		return err
	}

	enabled := parseEnabled(metadata)
	enableSearchDomain := parseEnableSearchDomain(metadata)

	// Check if zone already exists
	for _, zone := range zones {
		if zone.Domain == domain {
			// Zone exists, check if we need to update settings
			// Only update if metadata explicitly specifies a value that differs
			if (enabled != nil && zone.Enabled != *enabled) ||
				(enableSearchDomain != nil && zone.EnableSearchDomain != *enableSearchDomain) {
				// Update the zone settings, keeping unspecified fields as-is
				req := Zone{
					ID:                 zone.ID,
					Name:               zone.Name,
					Domain:             zone.Domain,
					Enabled:            zone.Enabled,
					EnableSearchDomain: zone.EnableSearchDomain,
					DistributionGroups: zone.DistributionGroups,
				}
				if enabled != nil {
					req.Enabled = *enabled
				}
				if enableSearchDomain != nil {
					req.EnableSearchDomain = *enableSearchDomain
				}
				return api.updateZone(zone.ID, &req)
			}
			return nil
		}
	}

	// Create the zone with specified values or defaults
	req := Zone{
		Name:               domain,
		Domain:             domain,
		Enabled:            true,  // Default for new zones
		EnableSearchDomain: false, // Default for new zones
		DistributionGroups: []string{},
	}
	if enabled != nil {
		req.Enabled = *enabled
	}
	if enableSearchDomain != nil {
		req.EnableSearchDomain = *enableSearchDomain
	}

	return api.createZone(&req)
}

// ListZones returns the list of zones (domains) in this account.
func (api *netbirdProvider) ListZones() ([]string, error) {
	zones, err := api.listZones()
	if err != nil {
		return nil, err
	}

	var result []string
	for _, zone := range zones {
		result = append(result, zone.Domain)
	}
	return result, nil
}

// GetNameservers returns the nameservers for domain.
// NetBird doesn't provide traditional nameservers as it's a peer-to-peer DNS service.
func (api *netbirdProvider) GetNameservers(_ string) ([]*models.Nameserver, error) {
	// NetBird doesn't have traditional nameservers
	return nil, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *netbirdProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	zone, err := api.findZoneByDomain(domain)
	if err != nil {
		return nil, err
	}

	// Cache the zone ID for later use in corrections
	api.zoneMu.Lock()
	api.zoneMap[domain] = &zoneInfo{
		id:     zone.ID,
		domain: zone.Domain,
	}
	api.zoneMu.Unlock()

	// Get records for the zone
	records, err := api.listRecords(zone.ID)
	if err != nil {
		return nil, err
	}

	var existingRecords []*models.RecordConfig
	for _, r := range records {
		rc, err := nativeToRecordConfig(domain, &r)
		if err != nil {
			return nil, err
		}
		existingRecords = append(existingRecords, rc)
	}

	return existingRecords, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *netbirdProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction

	// Check if zone settings need to be updated
	zone, err := api.findZoneByDomain(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	// Parse metadata for zone settings
	enabled := parseEnabled(dc.Metadata)
	enableSearchDomain := parseEnableSearchDomain(dc.Metadata)

	// Build update request if any settings need to change
	if enabled != nil || enableSearchDomain != nil {
		// Check if values actually differ
		if (enabled == nil || zone.Enabled == *enabled) &&
			(enableSearchDomain == nil || zone.EnableSearchDomain == *enableSearchDomain) {
			// No changes needed
		} else {
			zoneID := zone.ID
			var parts []string
			if enabled != nil && zone.Enabled != *enabled {
				if *enabled {
					parts = append(parts, "enabled")
				} else {
					parts = append(parts, "disabled")
				}
			}
			if enableSearchDomain != nil && zone.EnableSearchDomain != *enableSearchDomain {
				if *enableSearchDomain {
					parts = append(parts, "search domain enabled")
				} else {
					parts = append(parts, "search domain disabled")
				}
			}

			corrections = append(corrections, &models.Correction{
				Msg: fmt.Sprintf("Update zone settings: %s", strings.Join(parts, ", ")),
				F: func() error {
					currentZone, err := api.findZoneByDomain(dc.Name)
					if err != nil {
						return err
					}
					req := Zone{
						ID:                 currentZone.ID,
						Name:               currentZone.Name,
						Domain:             currentZone.Domain,
						Enabled:            currentZone.Enabled,
						EnableSearchDomain: currentZone.EnableSearchDomain,
						DistributionGroups: currentZone.DistributionGroups,
					}
					if enabled != nil {
						req.Enabled = *enabled
					}
					if enableSearchDomain != nil {
						req.EnableSearchDomain = *enableSearchDomain
					}
					return api.updateZone(zoneID, &req)
				},
			})
		}
	}

	instructions, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	// Get zone ID from cache for use in corrections
	api.zoneMu.Lock()
	cachedZone, ok := api.zoneMap[dc.Name]
	api.zoneMu.Unlock()
	if !ok {
		return nil, 0, fmt.Errorf("zone not found in cache for domain: %s (was GetZoneRecords called?)", dc.Name)
	}
	zoneID := cachedZone.id

	addCorrection := func(msg string, f func() error) {
		corrections = append(corrections,
			&models.Correction{
				Msg: msg,
				F:   f,
			})
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
			req := recordConfigToNative(inst.New[0], dc.Name)
			addCorrection(inst.MsgsJoined, func() error {
				return api.createRecord(zoneID, req)
			})

		case diff2.CHANGE:
			id := inst.Old[0].Original.(*Record).ID
			req := recordConfigToNative(inst.New[0], dc.Name)
			addCorrection(inst.MsgsJoined, func() error {
				return api.updateRecord(zoneID, id, req)
			})

		case diff2.DELETE:
			id := inst.Old[0].Original.(*Record).ID
			addCorrection(inst.MsgsJoined, func() error {
				return api.deleteRecord(zoneID, id)
			})

		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

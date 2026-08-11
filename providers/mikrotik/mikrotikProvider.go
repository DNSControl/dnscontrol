package mikrotik

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

/*
MikroTik RouterOS DNS provider

Manages DNS static entries on a MikroTik RouterOS device via the REST API.

Info required in creds.json:
   - host      (RouterOS REST API endpoint, e.g. "http://192.168.88.1:8080")
   - username
   - password

RouterOS DNS is a flat list of static entries (no zone concept).
The provider filters records by the domain suffix to emulate zones.

Supported record types: A, AAAA, CNAME, MX, NS, SRV, TXT
Custom record type: MIKROTIK_FWD (RouterOS FWD entries for conditional forwarding)
*/

var features = providers.DocumentationNotes{
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanConcur:              providers.Cannot(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseCAA:              providers.Cannot(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseDSForChildren:    providers.Cannot(),
	providers.CanUseHTTPS:            providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseSVCB:             providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Cannot(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "MIKROTIK"
	const providerMaintainer = "@hedger"
	fns := providers.DspFuncs{
		Initializer:   newMikrotikProvider,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterCustomRecordType("MIKROTIK_FWD", providerName, "")
	providers.RegisterCustomRecordType("MIKROTIK_NXDOMAIN", providerName, "")
	providers.RegisterCustomRecordType("MIKROTIK_FORWARDER", providerName, "")
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "MikroTik RouterOS",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/mikrotik",
		PortalURL:   "", // No portal; managed on-device
		Fields: []providers.CredsField{
			{
				Key:      "host",
				Label:    "Host URL",
				Help:     "The MikroTik RouterOS REST API endpoint (for example http://192.168.88.1:8080).",
				Required: true,
			},
			{
				Key:      "username",
				Label:    "Username",
				Help:     "RouterOS API username.",
				Required: true,
			},
			{
				Key:      "password",
				Label:    "Password",
				Help:     "RouterOS API password.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

func newMikrotikProvider(cfg map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	host := cfg["host"]
	username := cfg["username"]
	password := cfg["password"]

	if host == "" {
		return nil, fmt.Errorf("mikrotik: 'host' is required")
	}
	if username == "" {
		return nil, fmt.Errorf("mikrotik: 'username' is required")
	}
	if password == "" {
		return nil, fmt.Errorf("mikrotik: 'password' is required")
	}

	host = strings.TrimRight(host, "/")

	p := &mikrotikProvider{
		host:     host,
		username: username,
		password: password,
	}

	// Optional comma-separated list of zones to help ListZones() identify
	// zones with 3+ labels (e.g. "internal.corp.local,home.arpa").
	if hints := cfg["zonehints"]; hints != "" {
		for h := range strings.SplitSeq(hints, ",") {
			h = strings.TrimSpace(h)
			if h != "" {
				p.zoneHints = append(p.zoneHints, h)
			}
		}
		// Sort longest-first so that more specific zones match before shorter ones.
		sort.Slice(p.zoneHints, func(i, j int) bool {
			return len(p.zoneHints[i]) > len(p.zoneHints[j])
		})
	}

	return p, nil
}

// ListZones enumerates zones by fetching all static DNS records and grouping
// their names by effective second-level domain (e.g. "host.example.com" → "example.com").
// RouterOS has no native zone concept, so this is an approximation.
//
// If "zonehints" is configured in creds.json (comma-separated list of zone names),
// records are first matched against those hints (longest match wins). This enables
// correct zone detection for multi-label private zones like "internal.corp.local".
func (p *mikrotikProvider) ListZones() ([]string, error) {
	nativeRecords, err := p.getAllRecords()
	if err != nil {
		return nil, fmt.Errorf("mikrotik: failed to list records: %w", err)
	}

	seen := map[string]bool{}

	// Always include all configured zone hints so that new (empty) zones
	// pass the zone-existence check in the framework.
	for _, hint := range p.zoneHints {
		seen[hint] = true
	}

	for _, nr := range nativeRecords {
		if nr.Dynamic == "true" || nr.Disabled == "true" {
			continue
		}
		zone := p.detectZone(nr.Name)
		seen[zone] = true
	}

	zones := make([]string, 0, len(seen)+1)

	// Prepend the synthetic forwarder zone so that get-zones outputs it
	// before regular zones. This ensures forwarder entries (referenced by
	// name in MIKROTIK_FWD targets) are created before the zones that use them.
	fwds, err := p.getAllForwarders()
	if err == nil && len(fwds) > 0 {
		zones = append(zones, ForwarderZone)
	}

	for z := range seen {
		zones = append(zones, z)
	}
	sort.Strings(zones[len(zones)-len(seen):]) // sort only the regular zones

	return zones, nil
}

// GetNameservers returns an empty list since RouterOS static DNS does not expose nameservers.
func (p *mikrotikProvider) GetNameservers(_ string) ([]*models.Nameserver, error) {
	return []*models.Nameserver{}, nil
}

// EnsureZoneExists is a no-op for RouterOS. Zones are virtual constructs
// derived from record names — the zone will "exist" once records are pushed.
func (p *mikrotikProvider) EnsureZoneExists(_ *models.DomainConfig) error {
	return nil
}

// GetZoneRecords fetches all static DNS records from RouterOS and filters by domain.
// For the special zone "_forwarders.mikrotik", it returns DNS forwarder entries instead.
func (p *mikrotikProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	if domain == ForwarderZone {
		return p.getForwarderRecords()
	}

	nativeRecords, err := p.getAllRecords()
	if err != nil {
		return nil, fmt.Errorf("mikrotik: failed to list records: %w", err)
	}

	var records models.Records
	for _, nr := range nativeRecords {
		if nr.Dynamic == "true" {
			continue
		}
		if nr.Disabled == "true" {
			continue
		}
		if !belongsToDomain(nr.Name, domain) {
			continue
		}
		// When zone hints are configured, a record like "host.h.example.com"
		// matches both "h.example.com" and "example.com" via suffix check. Use
		// detectZone to assign each record to exactly one zone.
		if len(p.zoneHints) > 0 && p.detectZone(nr.Name) != domain {
			continue
		}

		rcs, err := nativeToRecords(nr, domain)
		if err != nil {
			printer.Warnf("mikrotik: skipping record %q (type=%s): %v\n", nr.Name, nr.Type, err)
			continue
		}
		records = append(records, rcs...)
	}

	return records, nil
}

func (p *mikrotikProvider) getForwarderRecords() (models.Records, error) {
	fwds, err := p.getAllForwarders()
	if err != nil {
		return nil, fmt.Errorf("mikrotik: failed to list forwarders: %w", err)
	}

	var records models.Records
	for _, fwd := range fwds {
		if fwd.Disabled == "true" {
			continue
		}
		records = append(records, forwarderToRecord(fwd))
	}
	return records, nil
}

// GetZoneRecordsCorrections computes and returns corrections.
func (p *mikrotikProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	if dc.Name == ForwarderZone {
		return p.getForwarderCorrections(dc, existingRecords)
	}

	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, metaCompFunc)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, change := range changes {
		var corr *models.Correction
		switch change.Type {
		case diff2.REPORT:
			corr = &models.Correction{Msg: change.MsgsJoined}

		case diff2.CREATE:
			newRec := change.New[0]
			msg := change.MsgsJoined
			corr = &models.Correction{
				Msg: msg,
				F: func() error {
					native, err := recordToNative(newRec)
					if err != nil {
						return err
					}
					return p.createRecord(native)
				},
			}

		case diff2.CHANGE:
			oldRec := change.Old[0]
			newRec := change.New[0]
			msg := change.MsgsJoined
			corr = &models.Correction{
				Msg: msg,
				F: func() error {
					oldOriginal, ok := oldRec.Original.(*dnsStaticRecord)
					if !ok {
						return fmt.Errorf("mikrotik: missing original record data for update")
					}
					native, err := recordToNative(newRec)
					if err != nil {
						return err
					}
					return p.updateRecord(oldOriginal.ID, native)
				},
			}

		case diff2.DELETE:
			oldRec := change.Old[0]
			msg := change.MsgsJoined
			corr = &models.Correction{
				Msg: msg,
				F: func() error {
					oldOriginal, ok := oldRec.Original.(*dnsStaticRecord)
					if !ok {
						return fmt.Errorf("mikrotik: missing original record data for delete")
					}
					return p.deleteRecord(oldOriginal.ID)
				},
			}

		default:
			panic(fmt.Sprintf("mikrotik: unhandled change type: %s", change.Type))
		}

		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}

func belongsToDomain(fqdn, domain string) bool {
	if fqdn == domain {
		return true
	}
	return strings.HasSuffix(fqdn, "."+domain)
}

// detectZone determines which zone a record name belongs to.
// It first checks configured zonehints (longest match wins), then falls back
// to publicsuffix.EffectiveTLDPlusOne, and finally to the last two labels.
func (p *mikrotikProvider) detectZone(name string) string {
	// Try zone hints first (already sorted longest-first).
	for _, hint := range p.zoneHints {
		if belongsToDomain(name, hint) {
			return hint
		}
	}

	// Public suffix lookup.
	zone, err := publicsuffix.EffectiveTLDPlusOne(name)
	if err == nil {
		return zone
	}

	// Fallback: take the last two labels.
	parts := strings.Split(name, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return name
}

// forwarderCompFunc returns extra comparison data for forwarder records.
func forwarderCompFunc(rc *models.RecordConfig) string {
	if rc.Metadata == nil {
		return ""
	}
	return fmt.Sprintf("doh_servers=%s verify_doh_cert=%s",
		rc.Metadata["doh_servers"],
		rc.Metadata["verify_doh_cert"],
	)
}

// getForwarderCorrections computes corrections for the _forwarders.mikrotik zone.
func (p *mikrotikProvider) getForwarderCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, forwarderCompFunc)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, change := range changes {
		var corr *models.Correction
		switch change.Type {
		case diff2.REPORT:
			corr = &models.Correction{Msg: change.MsgsJoined}

		case diff2.CREATE:
			newRec := change.New[0]
			msg := change.MsgsJoined
			corr = &models.Correction{
				Msg: msg,
				F: func() error {
					f := recordToForwarder(newRec)
					return p.createForwarder(f)
				},
			}

		case diff2.CHANGE:
			oldRec := change.Old[0]
			newRec := change.New[0]
			msg := change.MsgsJoined
			corr = &models.Correction{
				Msg: msg,
				F: func() error {
					oldFwd, ok := oldRec.Original.(*dnsForwarder)
					if !ok {
						return fmt.Errorf("mikrotik: missing original forwarder data for update")
					}
					f := recordToForwarder(newRec)
					return p.updateForwarder(oldFwd.ID, f)
				},
			}

		case diff2.DELETE:
			oldRec := change.Old[0]
			msg := change.MsgsJoined
			corr = &models.Correction{
				Msg: msg,
				F: func() error {
					oldFwd, ok := oldRec.Original.(*dnsForwarder)
					if !ok {
						return fmt.Errorf("mikrotik: missing original forwarder data for delete")
					}
					return p.deleteForwarder(oldFwd.ID)
				},
			}

		default:
			panic(fmt.Sprintf("mikrotik: unhandled forwarder change type: %s", change.Type))
		}

		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}

// metaCompFunc returns extra comparison data for RouterOS records so that
// changes to match_subdomain, regexp, address_list, and comment are detected
// by the diff engine.
func metaCompFunc(rc *models.RecordConfig) string {
	if rc.Metadata == nil {
		return ""
	}
	s := fmt.Sprintf("address_list=%s comment=%s match_subdomain=%s regexp=%s",
		rc.Metadata["address_list"],
		rc.Metadata["comment"],
		rc.Metadata["match_subdomain"],
		rc.Metadata["regexp"],
	)
	// Return empty string if no metadata is actually set (avoid spurious diffs).
	if s == "address_list= comment= match_subdomain= regexp=" {
		return ""
	}
	return s
}

package ionos

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/DNSControl/dnscontrol/v5/pkg/zonecache"
)

/*

IONOS hosting DNS API provider (https://developer.hosting.ionos.com/docs/dns),
not to be confused with the IONOS Cloud DNS API.

Info required in `creds.json`:
   - api_key: the "<prefix>.<secret>" key sent as the X-API-Key header

*/

const defaultBaseURL = "https://api.hosting.ionos.com/dns/v1"

// untestedType explains the record types the API lists but this provider does
// not enable: the API documentation does not describe their content format.
const untestedType = "Listed by the API, content format not validated yet"

func init() {
	providers.Register[*ionosProvider]("IONOS", providers.Definition{
		FriendlyName:    "IONOS",
		Maintainer:      "@n-rodriguez",
		VendorAPIDocURL: "https://developer.hosting.ionos.com/docs/dns",
		PortalURL:       "https://developer.hosting.ionos.com/keys",
		SupportedTypes: []string{
			"Basic8",
			"CERT:Unimplemented:" + untestedType,
			"DS:Unimplemented:" + untestedType,
			"HTTPS:Unimplemented:" + untestedType,
			"LOC:Unimplemented:" + untestedType,
			"OPENPGPKEY:Unimplemented:" + untestedType,
			"RP:Unimplemented:" + untestedType,
			"SMIMEA:Unimplemented:" + untestedType,
			"SSHFP:Unimplemented:" + untestedType,
			"SVCB:Unimplemented:" + untestedType,
			"TLSA:Unimplemented:" + untestedType,
			"URI:Unimplemented:" + untestedType,
		},
		CanDualHost: providers.Cannot("IONOS manages the apex NS records"),
		CredFields: []providers.CredsField{
			{
				Key:      "api_key",
				Label:    "API key",
				Help:     "Your IONOS hosting API key, in the form <prefix>.<secret>.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

// ionosProvider is the handle for operations on the IONOS hosting DNS API.
type ionosProvider struct {
	observer  providers.ConversionObserver
	client    *http.Client
	apiKey    string
	baseURL   string
	zoneCache zonecache.ZoneCache[zone]

	// sleep is time.Sleep, replaced in tests so that exercising the rate-limit
	// backoff does not cost the wall-clock time it is waiting out.
	sleep func(time.Duration)

	// skipped holds, per zone, the records GetZoneRecords could not parse.
	// Zones are read concurrently, hence the mutex.
	skippedMu sync.Mutex
	skipped   map[string][]record
}

// Initialize initializes a fresh provider instance.
func (p *ionosProvider) Initialize(m map[string]string, _ json.RawMessage, options *providers.CreateOptions) error {
	apiKey := m["api_key"]
	if apiKey == "" {
		return errors.New("missing IONOS api_key")
	}
	*p = ionosProvider{
		client:  &http.Client{Timeout: 30 * time.Second},
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		sleep:   time.Sleep,
	}
	p.zoneCache = zonecache.New(p.fetchAllZones)
	p.SetConversionObserver(options.WithDefaults().ConversionObserver)
	return nil
}

// SetConversionObserver installs the observer that records the record conversions.
func (p *ionosProvider) SetConversionObserver(observer providers.ConversionObserver) {
	p.observer = observer
}

// ListZones returns the names of the zones of the account.
func (p *ionosProvider) ListZones() ([]string, error) {
	return p.zoneCache.GetZoneNames()
}

// GetNameservers returns the enabled NS records at the apex of the zone: the
// API has no endpoint that lists the nameservers of a zone.
func (p *ionosProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	zoneID, err := p.zoneID(domain)
	if err != nil {
		return nil, err
	}
	records, err := p.listRecords(zoneID)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, r := range records {
		if r.Type == "NS" && !r.Disabled && strings.EqualFold(strings.TrimSuffix(r.Name, "."), domain) {
			names = append(names, strings.TrimSuffix(r.Content, "."))
		}
	}
	return models.ToNameservers(names)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (p *ionosProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	zoneID, err := p.zoneID(dc.Name)
	if err != nil {
		return nil, err
	}
	records, err := p.listRecords(zoneID)
	if err != nil {
		return nil, err
	}
	existing := make(models.Records, 0, len(records))
	var skipped []record
	for _, r := range records {
		if !isManaged(r, dc.Name) {
			continue
		}
		before := providers.BeginToRC(p.observer, "toRC", r)
		rc, err := toRC(dc, r)
		providers.EndToRC(p.observer, "toRC", before, r, models.Records{rc}, err)
		if err != nil {
			// A record entered outside DNSControl must not block the whole zone. Left
			// out of the existing records, it is neither rewritten nor deleted.
			printer.Warnf("IONOS: skipping a record of zone %s: %v\n", dc.Name, err)
			skipped = append(skipped, r)
			continue
		}
		existing = append(existing, rc)
	}
	p.rememberSkipped(dc.Name, skipped)
	return existing, nil
}

func (p *ionosProvider) rememberSkipped(domain string, records []record) {
	p.skippedMu.Lock()
	defer p.skippedMu.Unlock()
	if p.skipped == nil {
		p.skipped = map[string][]record{}
	}
	p.skipped[domain] = records
}

func (p *ionosProvider) skippedRecords(domain string) []record {
	p.skippedMu.Lock()
	defer p.skippedMu.Unlock()
	return p.skipped[domain]
}

// matchesSkipped tells whether a desired record has the same name, type and
// content as a record that could not be parsed. The quotes are ignored because
// an unparsable TXT content is typically one with unbalanced quotes.
func matchesSkipped(rc *models.RecordConfig, skipped []record) bool {
	native := toNative(rc)
	for _, r := range skipped {
		if r.Type == native.Type &&
			strings.EqualFold(strings.TrimSuffix(r.Name, "."), native.Name) &&
			strings.Trim(r.Content, `"`) == strings.Trim(native.Content, `"`) {
			return true
		}
	}
	return false
}

// isManaged tells whether a record belongs in the diff. IONOS manages the SOA and
// the apex NS records. The other unsupported types cannot be declared in
// dnsconfig.js, so diffing them would delete them on the first push.
func isManaged(r record, domain string) bool {
	switch r.Type {
	case "A", "AAAA", "CAA", "CNAME", "MX", "SRV", "TXT":
		return true
	case "NS":
		return !strings.EqualFold(strings.TrimSuffix(r.Name, "."), domain)
	default:
		return false
	}
}

// comparableDisabled makes a disabled record differ from its desired twin. IONOS
// deactivates records that conflict with its services, and such a record is not
// served even though its name, type and content match.
func comparableDisabled(rc *models.RecordConfig) string {
	if r, ok := rc.Original.(record); ok && r.Disabled {
		return "disabled=true"
	}
	return ""
}

// checkSkippedConflicts refuses a desired record that cannot coexist with a record
// that could not be parsed: a CNAME shares its name with no other record.
func checkSkippedConflicts(records models.Records, skipped []record) error {
	for _, rc := range records {
		name := rc.GetLabelFQDN()
		for _, r := range skipped {
			if strings.EqualFold(strings.TrimSuffix(r.Name, "."), name) && (r.Type == "CNAME" || rc.Type == "CNAME") {
				return fmt.Errorf("%s %s conflicts with the %s record at that name, which could not be parsed (%q): fix it in the IONOS control panel", rc.Type, name, r.Type, r.Content)
			}
		}
	}
	return nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (p *ionosProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	// The apex NS records are left out of the existing records, so they must not
	// be created again from the nameservers DNSControl adds to the desired zone.
	skipped := p.skippedRecords(dc.Name)
	dc.Filter(func(rc *models.RecordConfig) bool {
		if rc.Type == "NS" && rc.GetLabel() == "@" {
			return false
		}
		// Creating this record would duplicate the skipped one.
		if matchesSkipped(rc, skipped) {
			printer.Warnf("IONOS: leaving %s %s alone: it matches a record that could not be parsed\n", rc.Type, rc.GetLabelFQDN())
			return false
		}
		return true
	})
	if err := checkSkippedConflicts(dc.Records, skipped); err != nil {
		return nil, 0, err
	}

	changes, actualChangeCount, err := diff2.ByRecord(existing, dc, comparableDisabled)
	if err != nil {
		return nil, 0, err
	}
	if len(changes) == 0 {
		return nil, actualChangeCount, nil
	}

	zoneID, err := p.zoneID(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE:
			native := p.observedToNative(change.New[0])
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					return p.createRecord(zoneID, native)
				},
			})
		case diff2.CHANGE:
			native := p.observedToNative(change.New[0])
			id := change.Old[0].Original.(record).ID
			corrections = append(corrections, &models.Correction{
				Msg: fmt.Sprintf("%s; IONOS record ID: %s", change.Msgs[0], id),
				F: func() error {
					return p.updateRecord(zoneID, id, native)
				},
			})
		case diff2.DELETE:
			id := change.Old[0].Original.(record).ID
			corrections = append(corrections, &models.Correction{
				Msg: fmt.Sprintf("%s; IONOS record ID: %s", change.Msgs[0], id),
				F: func() error {
					return p.deleteRecord(zoneID, id)
				},
			})
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}
	return corrections, actualChangeCount, nil
}

// observedToNative converts a record sent to the API, reporting the conversion
// to the observer.
func (p *ionosProvider) observedToNative(rc *models.RecordConfig) record {
	input := models.Records{rc}
	before := providers.BeginToNative(p.observer, "toNative", input)
	native := toNative(rc)
	providers.EndToNative(p.observer, "toNative", before, input, native, nil)
	return native
}

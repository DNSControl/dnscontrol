// Package nameocean provides a provider for managing zones at NameOcean.
//
// The provider uses the NameOcean API v1 "DNS as code" endpoints: it exports
// the whole zone as one JSON document and applies the desired zone back with
// one atomic, serial-checked request. See https://nameocean.net/dashboard/api/
package nameocean

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
)

// defaultNameservers are the apex nameservers of every NameOcean zone. They
// are not part of the zone document and cannot be changed.
var defaultNameservers = []string{
	"ns1.nameocean.net",
	"ns2.nameocean.net",
	"ns3.nameocean.net",
	"ns4.nameocean.net",
}

// zoneState is what GetZoneRecords learned about a zone that the apply needs:
// the serial of the export, and the records this provider does not manage.
type zoneState struct {
	serial      string
	passthrough []json.RawMessage
}

// nameoceanProvider is the handle for this provider.
type nameoceanProvider struct {
	client   *http.Client
	apiURL   string
	token    string
	observer providers.ConversionObserver

	mu    sync.Mutex
	zones map[string]*zoneState
}

func init() {
	providers.Register[*nameoceanProvider]("NAMEOCEAN", providers.Definition{
		FriendlyName: "NameOcean",
		Maintainer:   "@akarca",
		SupportedTypes: []string{
			"Basic8",
			"ALIAS",
			"HTTPS",
			"SVCB",
			"PTR:Cannot:NameOcean keys PTR records by IP address, not by DNS label",
		},
		CanAutoDNSSEC:       providers.Unimplemented("NameOcean signs zones through its own DNSSEC API, which this provider does not use yet"),
		CanDualHost:         providers.Cannot("The apex NS records are always ns1-ns4.nameocean.net"),
		CanUseDSForChildren: providers.Can(),
		CredFields: []providers.CredsField{
			{
				Key:      "token",
				Label:    "API key",
				Help:     "A NameOcean API key with the dns:read and dns:write scopes. The account needs two-step login.",
				Secret:   true,
				Required: true,
				EnvVar:   "NAMEOCEAN_API_KEY",
			},
		},
		VendorAPIDocURL: "https://nameocean.net/dashboard/api/",
		PortalURL:       "https://nameocean.net/dashboard/api/",
		Notes:           "API keys need the dns:read and dns:write scopes, and the account needs two-step login enabled.",
	})
}

// SetConversionObserver sets the observer for record conversions.
func (p *nameoceanProvider) SetConversionObserver(observer providers.ConversionObserver) {
	p.observer = observer
}

// Initialize initializes a fresh provider instance.
func (p *nameoceanProvider) Initialize(conf map[string]string, _ json.RawMessage, options *providers.CreateOptions) error {
	if conf["token"] == "" {
		return errors.New("nameocean: missing token")
	}
	apiURL := strings.TrimSuffix(conf["apiurl"], "/")
	if apiURL == "" {
		apiURL = defaultAPIURL
	}
	*p = nameoceanProvider{
		client: &http.Client{Timeout: 60 * time.Second},
		apiURL: apiURL,
		token:  conf["token"],
		zones:  map[string]*zoneState{},
	}
	p.SetConversionObserver(options.WithDefaults().ConversionObserver)
	return nil
}

// GetNameservers returns the nameservers for a domain.
func (p *nameoceanProvider) GetNameservers(string) ([]*models.Nameserver, error) {
	return models.ToNameservers(defaultNameservers)
}

// ListZones returns the zones in the account.
func (p *nameoceanProvider) ListZones() ([]string, error) {
	return p.listZones()
}

// EnsureZoneExists creates a zone if it does not exist.
func (p *nameoceanProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	return p.createZone(dc.Name)
}

func (p *nameoceanProvider) setZoneState(zone string, state *zoneState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.zones[zone] = state
}

func (p *nameoceanProvider) getZoneState(zone string) *zoneState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.zones[zone]
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (p *nameoceanProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	doc, serial, err := p.exportZone(dc.Name)
	if err != nil {
		return nil, err
	}
	records, passthrough, err := p.recordsFromDocument(dc, doc)
	if err != nil {
		return nil, err
	}
	p.setZoneState(dc.Name, &zoneState{serial: serial, passthrough: passthrough})
	return records, nil
}

// recordsFromDocument converts the managed records of a zone document and
// returns the others (REDIRECT, PTR, ...) unchanged.
func (p *nameoceanProvider) recordsFromDocument(dc *models.DomainConfig, doc *zoneDocument) (models.Records, []json.RawMessage, error) {
	records := models.Records{}
	var passthrough []json.RawMessage
	for i, raw := range doc.Records {
		var native nativeRecord
		if err := json.Unmarshal(raw, &native); err != nil {
			return nil, nil, fmt.Errorf("nameocean: zone %q record %d: %w", dc.Name, i, err)
		}
		if !managedTypes[strings.ToUpper(native.Type)] {
			passthrough = append(passthrough, raw)
			continue
		}
		before := providers.BeginToRC(p.observer, "toRC", native)
		rc, err := toRC(dc, native)
		providers.EndToRC(p.observer, "toRC", before, native, models.Records{rc}, err)
		if err != nil {
			return nil, nil, err
		}
		records = append(records, rc)
	}
	return records, passthrough, nil
}

// prepareDesired drops the apex NS records (NameOcean serves its own and
// does not list them) and snaps each TTL to one NameOcean accepts.
func prepareDesired(dc *models.DomainConfig) {
	kept := make(models.Records, 0, len(dc.Records))
	for _, rc := range dc.Records {
		if rc.Type == "NS" && rc.GetLabel() == "@" {
			ns := strings.ToLower(strings.TrimSuffix(rc.AsNS().Ns, "."))
			if !strings.HasSuffix(ns, ".nameocean.net") {
				printer.Warnf("NAMEOCEAN: the apex NS records of %s are always ns1-ns4.nameocean.net; ignoring NS %s\n", dc.Name, ns)
			}
			continue
		}
		if fixed := fixTTL(rc.Type, rc.TTL); fixed != rc.TTL {
			printer.Warnf("NAMEOCEAN: %s %s: TTL %d is not supported, using %d\n", rc.Type, rc.GetLabelFQDN(), rc.TTL, fixed)
			rc.TTL = fixed
		}
		kept = append(kept, rc)
	}
	dc.Records = kept
}

// buildDocument converts the desired records into a zone document and adds
// the passthrough records the export returned.
func (p *nameoceanProvider) buildDocument(dc *models.DomainConfig, desired models.Records, passthrough []json.RawMessage) (*zoneDocument, error) {
	doc := &zoneDocument{Zone: dc.Name, Records: make([]json.RawMessage, 0, len(desired)+len(passthrough))}
	for _, rc := range desired {
		before := providers.BeginToNative(p.observer, "toNative", models.Records{rc})
		native, err := toNative(dc, rc)
		providers.EndToNative(p.observer, "toNative", before, models.Records{rc}, native, err)
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(native)
		if err != nil {
			return nil, err
		}
		doc.Records = append(doc.Records, raw)
	}
	doc.Records = append(doc.Records, passthrough...)
	return doc, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (p *nameoceanProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	prepareDesired(dc)

	result, err := diff2.ByZone(existing, dc, nil)
	if err != nil {
		return nil, 0, err
	}
	if !result.HasChanges {
		return nil, result.ActualChangeCount, nil
	}

	state := p.getZoneState(dc.Name)
	if state == nil {
		state = &zoneState{}
	}
	doc, err := p.buildDocument(dc, result.DesiredPlus, state.passthrough)
	if err != nil {
		return nil, 0, err
	}

	corrections := []*models.Correction{
		{
			Msg: strings.Join(result.Msgs, "\n"),
			F: func() error {
				_, err := p.applyZone(dc.Name, state.serial, doc)
				return err
			},
		},
	}
	return corrections, result.ActualChangeCount, nil
}

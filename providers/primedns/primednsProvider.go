package primedns

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

/*

PrimeDNS (https://primedns.net) DNS provider.

Info required in `creds.json`:
   - api_key   A PrimeDNS API key (X-API-Key). It needs the scopes dns:read and dns:write;
               and zones:create to let "dnscontrol create-zones" add a zone.
Optional:
   - api_url   The API base URL. Default https://api.primedns.net/v1

*/

func init() {
	providers.Register[*primednsProvider]("PRIMEDNS", providers.Definition{
		FriendlyName: "PrimeDNS",
		Maintainer:   "@voyo",
		SupportedTypes: []string{
			"Basic8",
			"ALIAS",
			"DNAME",
			"DS",
			"HTTPS",
			"LOC",
			"NAPTR",
			"OPENPGPKEY",
			"PTR",
			"SSHFP",
			"SVCB",
			"TLSA",
		},
		// The API allows 300 to 1000 requests per minute depending on the plan of the key's owner (plans.api_rate_limit),
		// unless the key itself sets a lower limit; see documentation/provider/primedns.md, "Rate limit".
		CanConcur: providers.Cannot(),
		CredFields: []providers.CredsField{
			{
				Key:      "api_key",
				Label:    "API key",
				Help:     "A PrimeDNS API key with the scopes dns:read and dns:write. get-zones with all also needs domains:read, create-zones needs zones:create.",
				Secret:   true,
				Required: true,
			},
			{
				Key:   "api_url",
				Label: "API URL",
				Help:  "The API base URL. Leave empty for https://api.primedns.net/v1.",
			},
		},
		PortalURL: "https://primedns.net/",
	})
}

// primednsProvider is the PrimeDNS DNSServiceProvider.
type primednsProvider struct {
	client  *http.Client
	apiKey  string
	baseURL string
}

// Initialize initializes a fresh provider instance.
func (p *primednsProvider) Initialize(m map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	apiKey := m["api_key"]
	if apiKey == "" {
		return errors.New("missing PrimeDNS api_key")
	}
	baseURL := strings.TrimRight(m["api_url"], "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	*p = primednsProvider{
		client:  &http.Client{Timeout: 60 * time.Second},
		apiKey:  apiKey,
		baseURL: baseURL,
	}
	return nil
}

// GetNameservers returns no nameservers: PrimeDNS manages the apex NS records itself. Delegation at the
// registrar is done with NAMESERVER("ns1.primedns.net") and NAMESERVER("ns2.primedns.net") if you want it automated.
func (p *primednsProvider) GetNameservers(string) ([]*models.Nameserver, error) {
	return nil, nil
}

// ListZones returns the zones of the account (used by "dnscontrol get-zones").
func (p *primednsProvider) ListZones() ([]string, error) {
	zones, err := p.listZones()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(zones))
	for _, z := range zones {
		names = append(names, strings.TrimSuffix(z.Name, "."))
	}
	return names, nil
}

// EnsureZoneExists adds the zone to the account when it is missing.
func (p *primednsProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	records, err := p.listRecords(dc.Name)
	if err != nil {
		return err
	}
	if len(records) > 0 {
		return nil
	}
	return p.createZone(dc.Name)
}

// GetZoneRecords returns the records of the zone.
func (p *primednsProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	records, err := p.listRecords(dc.Name)
	if err != nil {
		return nil, fmt.Errorf("listing the records of %s: %w", dc.Name, err)
	}
	// A zone of the account always has its SOA and NS records. The API answers an empty list both for a zone that
	// is not in the account (or not allowed for this key) and for a zone that does not exist.
	if len(records) == 0 {
		return nil, fmt.Errorf("zone %s: %w", dc.Name, errZoneNotFound)
	}

	cur := make(models.Records, 0, len(records))
	for _, r := range records {
		if skipRecord(dc.Name, r) {
			continue
		}
		rc, err := toRecordConfig(dc, r)
		if err != nil {
			return nil, fmt.Errorf("record %s %s: %w", r.Name, r.Type, err)
		}
		cur = append(cur, rc)
	}
	return cur, nil
}

// GetZoneRecordsCorrections returns the changes that turn curRecords into the desired records of dc. PrimeDNS
// replaces a whole RRset (all records of one type at one name) in one atomic request, so a change of an RRset is
// one PUT and its removal one DELETE.
func (p *primednsProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, curRecords models.Records) ([]*models.Correction, int, error) {
	changes, actualChangeCount, err := diff2.ByRecordSet(curRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, change := range changes {
		zone, typ := dc.Name, change.Key.Type
		var name string // the short label, "@" for the apex
		switch {
		case len(change.New) > 0:
			name = change.New[0].GetLabel()
		case len(change.Old) > 0:
			name = change.Old[0].GetLabel()
		}
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE, diff2.CHANGE:
			in := toRRsetInput(change.New)
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F:   func() error { return p.putApexSafe(zone, name, typ, in) },
			})
		case diff2.DELETE:
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F:   func() error { return p.deleteApexSafe(zone, name, typ) },
			})
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}
	return corrections, actualChangeCount, nil
}

// toRRsetInput builds the PUT body from the desired records of one RRset.
func toRRsetInput(rcs models.Records) rrsetInput {
	in := rrsetInput{TTL: int(rcs[0].TTL)}
	for _, rc := range rcs {
		r := fromRecordConfig(rc)
		in.Records = append(in.Records, rrsetValue{Content: r.Content, Prio: r.Prio})
	}
	return in
}

// ownApexNS returns the apex NS values that PrimeDNS manages itself and DNSControl does not see (skipRecord).
// A PUT replaces the whole RRset, so they have to be sent along with the NS records of the user.
func (p *primednsProvider) ownApexNS(zone, name, typ string) ([]rrsetValue, error) {
	if name != "@" || typ != "NS" {
		return nil, nil
	}
	cur, err := p.getRRset(zone, name, typ)
	if err != nil {
		var ae *apiError
		if errors.As(err, &ae) && ae.Status == 404 {
			return nil, nil
		}
		return nil, err
	}
	var own []rrsetValue
	for _, v := range cur {
		if skipRecord(zone, record{Name: zone, Type: "NS", Content: v.Content}) {
			own = append(own, v)
		}
	}
	return own, nil
}

func (p *primednsProvider) putApexSafe(zone, name, typ string, in rrsetInput) error {
	own, err := p.ownApexNS(zone, name, typ)
	if err != nil {
		return err
	}
	in.Records = append(own, in.Records...)
	return p.putRRset(zone, name, typ, in)
}

func (p *primednsProvider) deleteApexSafe(zone, name, typ string) error {
	own, err := p.ownApexNS(zone, name, typ)
	if err != nil {
		return err
	}
	if len(own) == 0 {
		return p.deleteRRset(zone, name, typ)
	}
	// Only the NS records of PrimeDNS stay; their TTL is kept as the zone has it.
	cur, err := p.getRRsetTTL(zone, name, typ)
	if err != nil {
		return err
	}
	return p.putRRset(zone, name, typ, rrsetInput{TTL: cur, Records: own})
}

package luadns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	dnsv2 "codeberg.org/miekg/dns"
	api "github.com/luadns/luadns-go"
	"golang.org/x/time/rate"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

/*

LuaDNS API DNS provider:

Info required in `creds.json`:
   - email
   - apikey
*/

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseOPENPGPKEY:       providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "LUADNS"
	const providerMaintainer = "@riku22"
	fns := providers.DspFuncs{
		Initializer:   NewLuaDNS,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "LuaDNS",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/luadns",
		PortalURL:   "https://app.luadns.com/users/api_keys",
		Fields: []providers.CredsField{
			{
				Key:      "email",
				Label:    "Email",
				Help:     "Your LuaDNS E-mail address.",
				Required: true,
			},
			{
				Key:      "apikey",
				Label:    "API key",
				Help:     "Specify the API key you created.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

type luadnsProvider struct {
	observer    providers.ConversionObserver
	provider    *api.Client
	ctx         context.Context
	rateLimiter *rate.Limiter
	nameServers []string
	zones       []*api.Zone
}

func (l *luadnsProvider) SetConversionObserver(observer providers.ConversionObserver) {
	l.observer = observer
}

// NewLuaDNS creates the provider.
func NewLuaDNS(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	if m["email"] == "" || m["apikey"] == "" {
		return nil, errors.New("missing LuaDNS email or apikey")
	}
	ctx := context.Background()
	rateLimiter := rate.NewLimiter(4, 1)
	provider := api.NewClient(m["email"], m["apikey"])
	user, err := provider.Me(ctx)
	if err != nil {
		return nil, err
	}
	l := &luadnsProvider{
		provider:    provider,
		ctx:         ctx,
		rateLimiter: rateLimiter,
		nameServers: user.NameServers,
	}
	return l, nil
}

// GetNameservers returns the nameservers for a domain.
func (l *luadnsProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameserversStripTD(l.nameServers)
}

// ListZones returns a list of the DNS zones.
func (l *luadnsProvider) ListZones() ([]string, error) {
	if err := l.fetchDomainList(); err != nil {
		return nil, err
	}
	zoneList := make([]string, 0, len(l.zones))
	for _, d := range l.zones {
		zoneList = append(zoneList, d.Name)
	}
	return zoneList, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (l *luadnsProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	zone, err := l.getZone(domain)
	if err != nil {
		return nil, err
	}
	records, err := l.getRecords(zone)
	if err != nil {
		return nil, err
	}
	existingRecords := make(models.Records, len(records))
	for i := range records {
		before := providers.BeginToRC(l.observer, "nativeToRecord", records[i])
		newr, err := nativeToRecord(dc, records[i])
		providers.EndToRC(l.observer, "nativeToRecord", before, records[i], models.Records{newr}, err)
		if err != nil {
			return nil, err
		}
		existingRecords[i] = newr
	}
	return existingRecords, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (l *luadnsProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, records models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction

	l.checkNS(dc)

	zone, err := l.getZone(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	changes, actualChangeCount, err := diff2.ByRecordSet(records, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE:
			before := providers.BeginToNative(l.observer, "recordsToNative", change.New)
			req := recordsToNative(change.New)
			providers.EndToNative(l.observer, "recordsToNative", before, change.New, req, nil)
			corrections = append(corrections, &models.Correction{
				F: func() error {
					if err := l.rateLimiter.Wait(l.ctx); err != nil {
						return err
					}
					_, err := l.provider.CreateManyRecords(l.ctx, zone, req)
					return err
				},
				Msg: change.MsgsJoined,
			})
		case diff2.CHANGE:
			before := providers.BeginToNative(l.observer, "recordsToNative", change.New)
			req := recordsToNative(change.New)
			providers.EndToNative(l.observer, "recordsToNative", before, change.New, req, nil)
			corrections = append(corrections, &models.Correction{
				F: func() error {
					if err := l.rateLimiter.Wait(l.ctx); err != nil {
						return err
					}
					_, err := l.provider.UpdateManyRecords(l.ctx, zone, req)
					return err
				},
				Msg: change.MsgsJoined,
			})
		case diff2.DELETE:
			before := providers.BeginToNative(l.observer, "recordsToNative", change.Old)
			req := recordsToNative(change.Old)
			providers.EndToNative(l.observer, "recordsToNative", before, change.Old, req, nil)
			corrections = append(corrections, &models.Correction{
				F: func() error {
					if err := l.rateLimiter.Wait(l.ctx); err != nil {
						return err
					}
					_, err := l.provider.DeleteManyRecords(l.ctx, zone, req)
					return err
				},
				Msg: change.MsgsJoined,
			})
		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", change.Type))
		}
	}
	return corrections, actualChangeCount, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (l *luadnsProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	if l.zones == nil {
		if err := l.fetchDomainList(); err != nil {
			return err
		}
	}
	if err := l.rateLimiter.Wait(l.ctx); err != nil {
		return err
	}
	zone, err := l.provider.CreateZone(l.ctx, &api.Zone{Name: domain})
	if err != nil {
		return err
	}
	l.zones = append(l.zones, zone)
	return nil
}

func (l *luadnsProvider) fetchDomainList() error {
	if err := l.rateLimiter.Wait(l.ctx); err != nil {
		return err
	}
	zones, err := l.provider.ListZones(l.ctx, &api.ListParams{})
	if err != nil {
		return err
	}
	l.zones = zones
	return nil
}

func (l *luadnsProvider) getZone(name string) (*api.Zone, error) {
	if l.zones == nil {
		if err := l.fetchDomainList(); err != nil {
			return nil, err
		}
	}
	for i := range l.zones {
		if l.zones[i].Name == name {
			return l.zones[i], nil
		}
	}
	return nil, fmt.Errorf("'%s' not a zone in luadns account", name)
}

func (l *luadnsProvider) getRecords(zone *api.Zone) ([]*api.Record, error) {
	if err := l.rateLimiter.Wait(l.ctx); err != nil {
		return nil, err
	}
	records, err := l.provider.ListRecords(l.ctx, zone, &api.ListParams{})
	if err != nil {
		return nil, err
	}
	var newRecords []*api.Record
	for _, rec := range records {
		if rec.Type == "SOA" {
			continue
		}
		newRecords = append(newRecords, rec)
	}
	return newRecords, nil
}

func (l *luadnsProvider) checkNS(dc *models.DomainConfig) {
	for _, rec := range dc.Records {
		// LuaDNS does not support changing the TTL of the default nameservers, so forcefully change the TTL to 86400.
		if rec.Type == "NS" && rec.Name == "@" && slices.Contains(l.nameServers, rec.GetRDATA().String()) && rec.TTL != 86400 {
			rec.TTL = 86400
		}
	}
}

func nativeToRecord(dc *models.DomainConfig, r *api.Record) (*models.RecordConfig, error) {
	label := dc.ToShort(r.Name)
	ttl := r.TTL
	var rc *models.RecordConfig
	var err error
	switch rtype := r.Type; rtype {
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Content)
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, r.Content)
	}
	if err == nil {
		rc.Original = r
	}
	return rc, err
}

func recordsToNative(rc models.Records) []*api.RR {
	var rrs []*api.RR
	for _, rec := range rc {
		r := &api.RR{
			Name: rec.GetLabelFQDN() + ".",
			Type: rec.Type,
			TTL:  rec.TTL,
		}
		switch rtype := rec.TypeNum; rtype {
		case dnsv2.TypeTXT:
			r.Content = rec.GetTargetTXTJoined()
		case dnsv2.TypeHTTPS:
			// The RDATA's String() quotes every SvcParam value (e.g. port="80",
			// alpn="h2,h3"), which LuaDNS's strict SVCB parser rejects. Build the
			// content with unquoted params (port=80, alpn=h2,h3) instead.
			f := rec.AsHTTPS()
			params := models.Svcbv2ValueToString(f.Value)
			if params == "" {
				r.Content = fmt.Sprintf("%d %s", f.Priority, f.Target)
			} else {
				r.Content = fmt.Sprintf("%d %s %s", f.Priority, f.Target, params)
			}
		default:
			r.Content = rec.GetRDATA().String()
		}
		rrs = append(rrs, r)
	}
	return rrs
}

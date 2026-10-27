package luadns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

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

func init() {
	providers.Register[*luadnsProvider]("LUADNS", providers.Definition{
		FriendlyName: "LuaDNS",
		Maintainer:   "@riku22",
		SupportedTypes: []string{
			"Basic8",
			"ALIAS",
			"HTTPS",
			"OPENPGPKEY",
			"PTR",
			"SSHFP",
			"TLSA",
		},
		CredFields: []providers.CredsField{
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
		PortalURL: "https://app.luadns.com/users/api_keys",
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

// Initialize initializes a fresh provider instance.
func (l *luadnsProvider) Initialize(m map[string]string, _ json.RawMessage, options *providers.CreateOptions) error {
	if m["email"] == "" || m["apikey"] == "" {
		return errors.New("missing LuaDNS email or apikey")
	}
	ctx := context.Background()
	rateLimiter := rate.NewLimiter(4, 1)
	provider := api.NewClient(m["email"], m["apikey"])
	user, err := provider.Me(ctx)
	if err != nil {
		return err
	}
	*l = luadnsProvider{
		provider:    provider,
		ctx:         ctx,
		rateLimiter: rateLimiter,
		nameServers: user.NameServers,
	}
	l.SetConversionObserver(options.WithDefaults().ConversionObserver)
	return nil
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

	// LuaDNS takes a list of records per request, so collect the changes by
	// kind and send each kind in as few requests as possible. Deletes go first,
	// then updates, then creates: diff2 orders a label's old records ahead of
	// the new ones that would conflict with them (A -> CNAME), and that order
	// must survive the regrouping.
	var deletes, updates, creates []pendingChange

	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE:
			creates = append(creates, l.toPending(change.New, change.MsgsJoined))
		case diff2.CHANGE:
			updates = append(updates, l.toPending(change.New, change.MsgsJoined))
		case diff2.DELETE:
			deletes = append(deletes, l.toPending(change.Old, change.MsgsJoined))
		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", change.Type))
		}
	}

	// Updates and deletes are safe to repeat after a 500. Creates are not: the
	// failed request may have been partly applied.
	corrections = append(corrections, l.batchCorrections(zone, deletes, l.provider.DeleteManyRecords, true)...)
	corrections = append(corrections, l.batchCorrections(zone, updates, l.provider.UpdateManyRecords, true)...)
	corrections = append(corrections, l.batchCorrections(zone, creates, l.provider.CreateManyRecords, false)...)
	return corrections, actualChangeCount, nil
}

// maxBatchRRs is the most RRs sent in one bulk request. LuaDNS documents no
// limit, so this is a conservative guess.
const maxBatchRRs = 100

// pendingChange is one diff2 change (the records of one recordset) in API form.
type pendingChange struct {
	rrs []*api.RR
	msg string
}

func (l *luadnsProvider) toPending(records models.Records, msg string) pendingChange {
	before := providers.BeginToNative(l.observer, "recordsToNative", records)
	rrs := recordsToNative(records)
	providers.EndToNative(l.observer, "recordsToNative", before, records, rrs, nil)
	return pendingChange{rrs: rrs, msg: msg}
}

// chunkChanges packs changes, in order, into requests of at most limit RRs.
// A change is never split: UpdateManyRecords replaces everything stored for
// each (name, type) it receives, so the records of one recordset must travel
// in the same request. A change larger than limit gets a request of its own.
func chunkChanges(changes []pendingChange, limit int) [][]pendingChange {
	var chunks [][]pendingChange
	var cur []pendingChange
	curRRs := 0
	for _, c := range changes {
		if len(cur) > 0 && curRRs+len(c.rrs) > limit {
			chunks = append(chunks, cur)
			cur, curRRs = nil, 0
		}
		cur = append(cur, c)
		curRRs += len(c.rrs)
	}
	if len(cur) > 0 {
		chunks = append(chunks, cur)
	}
	return chunks
}

type bulkFunc func(ctx context.Context, zone *api.Zone, rrs []*api.RR) ([]*api.Record, error)

// batchCorrections turns changes into one correction per bulk request.
func (l *luadnsProvider) batchCorrections(zone *api.Zone, changes []pendingChange, call bulkFunc, repeatable bool) []*models.Correction {
	var corrections []*models.Correction
	for _, chunk := range chunkChanges(changes, maxBatchRRs) {
		var rrs []*api.RR
		var msgs []string
		for _, c := range chunk {
			rrs = append(rrs, c.rrs...)
			msgs = append(msgs, c.msg)
		}
		send := func() error {
			if err := l.rateLimiter.Wait(l.ctx); err != nil {
				return err
			}
			_, err := call(l.ctx, zone, rrs)
			return err
		}
		corrections = append(corrections, &models.Correction{
			F: func() error {
				if repeatable {
					return retry500(send)
				}
				return send()
			},
			Msg: strings.Join(msgs, "\n"),
		})
	}
	return corrections
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

// retry500 runs fn, repeating it a few times if the API answers with a 500,
// which it occasionally does for a valid request. Only use it for requests
// that are safe to repeat (not creates).
func retry500(fn func() error) error {
	for attempt := 1; ; attempt++ {
		err := fn()
		if e, ok := errors.AsType[*api.ErrBadStatusCode](err); !ok || e.StatusCode != http.StatusInternalServerError || attempt == 3 {
			return err
		}
		time.Sleep(time.Duration(attempt) * 2 * time.Second)
	}
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

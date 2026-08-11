package digitalocean

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/digitalocean/godo"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
	"golang.org/x/oauth2"
)

/*

DigitalOcean API DNS provider:

Info required in `creds.json`:
   - token

*/

// digitaloceanProvider is the handle for operations.
type digitaloceanProvider struct {
	client *godo.Client
}

var defaultNameServerNames = []string{
	"ns1.digitalocean.com",
	"ns2.digitalocean.com",
	"ns3.digitalocean.com",
}

const perPageSize = 100

// NewDo creates a DO-specific DNS provider.
func NewDo(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	if m["token"] == "" {
		return nil, errors.New("no DigitalOcean token provided")
	}

	ctx := context.Background()
	oauthClient := oauth2.NewClient(
		ctx,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: m["token"]}),
	)
	client := godo.NewClient(oauthClient)

	api := &digitaloceanProvider{client: client}

	// Get a domain to validate the token
retry:
	_, resp, err := api.client.Domains.List(ctx, &godo.ListOptions{PerPage: 1})
	if err != nil {
		if pauseAndRetry(resp) {
			goto retry
		}
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("token for digitalocean is not valid")
	}

	return api, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanConcur:              providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseAlias:            providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanUseDNAME:            providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanUseDNSKEY:           providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanUseDS:               providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanUseHTTPS:            providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanUsePTR:              providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanUseSOA:              providers.Cannot("Technically SOA is supported but in reality the API only permits updates to the TTL. That is insufficient for DNSControl to claim 'support'"),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanUseSMIMEA:           providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanUseSVCB:             providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.CanUseTLSA:             providers.Cannot("Digital Ocean documents that this is not supported."),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "DIGITALOCEAN"
	const providerMaintainer = "@chicks-net"
	fns := providers.DspFuncs{
		Initializer:   NewDo,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "DigitalOcean",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/digitalocean",
		PortalURL:   "https://cloud.digitalocean.com/account/api/tokens",
		Fields: []providers.CredsField{
			{
				Key:      "token",
				Label:    "API token",
				Help:     "Your DigitalOcean personal access token.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

// EnsureZoneExists creates a zone if it does not exist.
func (api *digitaloceanProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
retry:
	ctx := context.Background()
	_, resp, err := api.client.Domains.Get(ctx, domain)
	if err != nil {
		if pauseAndRetry(resp) {
			goto retry
		}
		// return err
	}
	if resp.StatusCode == http.StatusNotFound {
		_, _, err := api.client.Domains.Create(ctx, &godo.DomainCreateRequest{
			Name:      domain,
			IPAddress: "",
		})
		return err
	}
	return err
}

// ListZones returns the list of zones (domains) in this account.
func (api *digitaloceanProvider) ListZones() ([]string, error) {
	ctx := context.Background()
	zones := []string{}
	opt := &godo.ListOptions{PerPage: perPageSize}
retry:
	for {
		result, resp, err := api.client.Domains.List(ctx, opt)
		if err != nil {
			if pauseAndRetry(resp) {
				goto retry
			}
			return nil, err
		}

		for _, d := range result {
			zones = append(zones, d.Name)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		opt.Page = page + 1
	}

	return zones, nil
}

// GetNameservers returns the nameservers for domain.
func (api *digitaloceanProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers(defaultNameServerNames)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *digitaloceanProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := getRecords(api, domain)
	if err != nil {
		return nil, err
	}

	var existingRecords []*models.RecordConfig
	for i := range records {
		r, err := toRc(domain, &records[i])
		if err != nil {
			return nil, err
		}
		if r.Type == "SOA" {
			continue
		}
		existingRecords = append(existingRecords, r)
	}

	return existingRecords, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *digitaloceanProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	ctx := context.Background()

	var corrections []*models.Correction

	instructions, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	addCorrection := func(msg string, f func() (*godo.Response, error)) {
		corrections = append(corrections,
			&models.Correction{
				Msg: msg,
				F: func() error {
				retry:
					resp, err := f()
					if err != nil {
						if pauseAndRetry(resp) {
							goto retry
						}
					}
					return err
				},
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
			req := toReq(inst.New[0])
			addCorrection(inst.MsgsJoined, func() (*godo.Response, error) {
				_, resp, err := api.client.Domains.CreateRecord(ctx, dc.Name, req)
				return resp, err
			})

		case diff2.CHANGE:
			id := inst.Old[0].Original.(*godo.DomainRecord).ID
			req := toReq(inst.New[0])
			addCorrection(inst.MsgsJoined, func() (*godo.Response, error) {
				_, resp, err := api.client.Domains.EditRecord(ctx, dc.Name, id, req)
				return resp, err
			})

		case diff2.DELETE:
			id := inst.Old[0].Original.(*godo.DomainRecord).ID
			addCorrection(inst.MsgsJoined, func() (*godo.Response, error) {
				return api.client.Domains.DeleteRecord(ctx, dc.Name, id)
			})

		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

func getRecords(api *digitaloceanProvider, name string) ([]godo.DomainRecord, error) {
	ctx := context.Background()

retry:

	records := []godo.DomainRecord{}
	opt := &godo.ListOptions{PerPage: perPageSize}
	for {
		result, resp, err := api.client.Domains.Records(ctx, name, opt)
		if err != nil {
			if pauseAndRetry(resp) {
				goto retry
			}
			return nil, err
		}

		records = append(records, result...)

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		opt.Page = page + 1
	}

	return records, nil
}

func toRc(domain string, r *godo.DomainRecord) (*models.RecordConfig, error) {
	// This handles "@" etc.
	name := dnsutilv1.AddOrigin(r.Name, domain)

	target := r.Data
	// Make target FQDN (#rtype_variations)
	if r.Type == "CNAME" || r.Type == "MX" || r.Type == "NS" || r.Type == "SRV" {
		// If target is the domainname, e.g. cname foo.example.com -> example.com,
		// DO returns "@" on read even if fqdn was written.
		switch target {
		case "@":
			target = domain
		case ".":
			target = ""
		}
		target = target + "."
	}

	t := &models.RecordConfig{
		Type:         r.Type,
		TTL:          uint32(r.TTL),
		MxPreference: uint16(r.Priority),
		SrvPriority:  uint16(r.Priority),
		SrvWeight:    uint16(r.Weight),
		SrvPort:      uint16(r.Port),
		Original:     r,
		CaaTag:       r.Tag,
		CaaFlag:      uint8(r.Flags),
	}
	t.SetLabelFromFQDN(name, domain)
	switch rtype := r.Type; rtype {
	case "TXT":
		if err := t.SetTargetTXT(target); err != nil {
			return nil, err
		}
	default:
		if err := t.SetTarget(target); err != nil {
			return nil, err
		}
	}
	return t, nil
}

func toReq(rc *models.RecordConfig) *godo.DomainRecordEditRequest {
	name := rc.GetLabel()         // DO wants the short name or "@" for apex.
	target := rc.GetTargetField() // DO uses the target field only for a single value
	priority := 0                 // DO uses the same property for MX and SRV priority

	switch rc.Type { // #rtype_variations
	case "CAA":
		// DO API requires that a CAA target ends in dot.
		// Interestingly enough, the value returned from API doesn't
		// contain a trailing dot.
		target = target + "."
	case "MX":
		priority = int(rc.MxPreference)
	case "SRV":
		priority = int(rc.SrvPriority)
	case "TXT":
		// TXT records are the one place where DO combines many items into one field.
		target = rc.GetTargetTXTJoined()
	default:
		// no action required
	}

	return &godo.DomainRecordEditRequest{
		Type:     rc.Type,
		Name:     name,
		Data:     target,
		TTL:      int(rc.TTL),
		Priority: priority,
		Port:     int(rc.SrvPort),
		Weight:   int(rc.SrvWeight),
		Tag:      rc.CaaTag,
		Flags:    int(rc.CaaFlag),
	}
}

// backoff is the amount of time to sleep if a 429 or 504 is received.
// It is doubled after each use.
var backoff = time.Second * 5

const maxBackoff = time.Minute * 3

func pauseAndRetry(resp *godo.Response) bool {
	statusCode := resp.StatusCode
	if statusCode != 429 && statusCode != 504 {
		backoff = time.Second * 5
		return false
	}

	// a simple exponential back-off with a 3-minute max.
	log.Printf("Delaying %v due to ratelimit\n", backoff)
	time.Sleep(backoff)
	backoff = min(backoff+(backoff/2), maxBackoff)
	return true
}

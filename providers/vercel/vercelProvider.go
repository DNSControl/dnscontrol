package vercel

/*
Vercel DNS provider (vercel.com)

Info required in `creds.json`:
	- team_id
	- api_token
*/

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	vercelClient "github.com/vercel/terraform-provider-vercel/client"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanGetZones:            providers.Cannot(),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanUseDNAME:            providers.Cannot(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Cannot(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseDSForChildren:    providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSVCB:             providers.Cannot(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.CanUseDNSKEY:           providers.Cannot(),
	providers.DocCreateDomains:       providers.Cannot("Vercel requires a domain to be associated with a project before it can be added and managed"),
	providers.DocDualHost:            providers.Cannot("Vercel does not allow sufficient control over the apex NS records"),
	providers.DocOfficiallySupported: providers.Cannot(),
}

// vercelProvider stores login credentials and represents and API connection.
type vercelProvider struct {
	observer providers.ConversionObserver
	client   vercelClient.Client
	apiToken string
	teamID   string

	createLimiter *rateLimiter
	updateLimiter *rateLimiter
	deleteLimiter *rateLimiter
	listLimiter   *rateLimiter
}

func (c *vercelProvider) SetConversionObserver(observer providers.ConversionObserver) {
	c.observer = observer
}

func init() {
	const providerName = "VERCEL"
	const providerMaintainer = "@SukkaW"
	fns := providers.DspFuncs{
		Initializer:   newProvider,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, providers.CanUseSRV, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

func newProvider(creds map[string]string, meta json.RawMessage) (providers.DNSServiceProvider, error) {
	if creds["api_token"] == "" {
		return nil, errors.New("api_token required for VERCEL")
	}

	c := vercelClient.New(
		creds["api_token"],
	)

	ctx := context.Background()

	team, err := c.Team(ctx, creds["team_id"])
	if err != nil {
		return nil, err
	}

	c = c.WithTeam(team)
	return &vercelProvider{
		client:   *c,
		apiToken: creds["api_token"],
		teamID:   creds["team_id"],
		// rate limiters
		createLimiter: newRateLimiter(100, time.Hour),
		updateLimiter: newRateLimiter(50, time.Minute),
		deleteLimiter: newRateLimiter(50, time.Minute),
		listLimiter:   newRateLimiter(50, time.Minute),
	}, nil
}

// GetNameservers returns empty array.
// Vercel doesn't permit apex NS records. Vercel's API doesn't even include apex NS records in their API response
// To prevent DNSControl from trying to create default NS records, let' return an empty array here, just like
// exoscale provider and gandi v5 provider.
func (c *vercelProvider) GetNameservers(_ string) ([]*models.Nameserver, error) {
	return []*models.Nameserver{}, nil
}

func (c *vercelProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	var zoneRecords models.Records

	records, err := c.ListDNSRecords(context.Background(), domain)
	if err != nil {
		return nil, err
	}

	for _, r := range records {
		// Vercel has some system-created records that can't be deleted/modified. They can be overridden
		// by creating new records (where the DNS will prefer your record), but those system records are
		// still included in the API response.
		//
		// Those records will have their "creator" being "system", some of them even has a comment field
		// "Vercel automatically manages this record. It may change without notice".
		//
		// Per https://github.com/DNSControl/dnscontrol/pull/3542#issuecomment-3560041419, let's
		// pretend those records don't exist, and diff2.ByRecord() will not affect these existing records.
		if r.Creator == "system" {
			continue
		}

		before := providers.BeginToRC(c.observer, "vercelRecordToRC", r)
		rc, err := vercelRecordToRC(dc, r)
		providers.EndToRC(c.observer, "vercelRecordToRC", before, r, models.Records{rc}, err)
		if err != nil {
			return nil, err
		}
		zoneRecords = append(zoneRecords, rc)
	}

	return zoneRecords, nil
}

func vercelRecordToRC(dc *models.DomainConfig, r DNSRecord) (*models.RecordConfig, error) {
	label := dc.LabelFromShort(r.Name)
	ttl := uint32(r.TTL)

	var rc *models.RecordConfig
	var err error
	switch rtype := r.RecordType; rtype {
	case "CNAME":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Value,
			nrc.Flags{TargetIsFqdnNoDot: true})
	case "MX":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.MXPriority, r.Value,
			nrc.Flags{TargetIsFqdnNoDot: true})
	case "SRV", "HTTPS", "SVCB":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Priority, r.Value,
			nrc.Flags{TargetIsFqdnNoDot: true, SrvWeirdSplit: true})
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, r.Value)
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, r.Value,
			nrc.Flags{TargetIsFqdnNoDot: true})
	}
	if err != nil {
		return nil, fmt.Errorf("unparsable %s record received from vercel: %w", r.RecordType, err)
	}

	rc.Original = r

	return rc, nil
}

func (c *vercelProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, records models.Records) ([]*models.Correction, int, error) {
	// Vercel is a "ByRecord" API.

	// Vercel enforces a minimum TTL of 60 seconds
	for _, record := range dc.Records {
		record.TTL = max(record.TTL, 60)
	}

	instructions, actualChangeCount, err := diff2.ByRecord(records, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, inst := range instructions {
		switch inst.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{
				Msg: inst.MsgsJoined,
			})
		case diff2.CREATE:
			corrections = append(corrections, c.mkCreateCorrection(dc.Name, inst.New[0], inst.Msgs[0]))
		case diff2.CHANGE:
			corrections = append(corrections, c.mkChangeCorrection(dc.Name, inst.Old[0], inst.New[0], inst.Msgs[0]))
		case diff2.DELETE:
			corrections = append(corrections, c.mkDeleteCorrection(dc.Name, inst.Old[0], inst.Msgs[0]))
		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

func (c *vercelProvider) mkCreateCorrection(domain string, newRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			ctx := context.Background()
			input := models.Records{newRec}
			before := providers.BeginToNative(c.observer, "toVercelCreateRequest", input)
			req, err := toVercelCreateRequest(domain, newRec)
			providers.EndToNative(c.observer, "toVercelCreateRequest", before, input, req, err)
			if err != nil {
				return err
			}
			_, err = c.CreateDNSRecord(ctx, req)
			return err
		},
	}
}

func (c *vercelProvider) mkChangeCorrection(domain string, oldRec, newRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			ctx := context.Background()
			existingID := oldRec.Original.(DNSRecord).ID

			// UpdateDNSRecord doesn't support type changes
			// If record type changed, delete and re-create
			if oldRec.Type != newRec.Type {
				// Delete old record
				if err := c.DeleteDNSRecord(ctx, domain, existingID); err != nil {
					return err
				}
				// re-create new record.
				// luckily, delete and create use different rate limit timers
				// thus we are most likely can go through both.
				input := models.Records{newRec}
				before := providers.BeginToNative(c.observer, "toVercelCreateRequest", input)
				req, err := toVercelCreateRequest(domain, newRec)
				providers.EndToNative(c.observer, "toVercelCreateRequest", before, input, req, err)
				if err != nil {
					return err
				}
				_, err = c.CreateDNSRecord(ctx, req)
				return err
			}

			input := models.Records{newRec}
			before := providers.BeginToNative(c.observer, "toVercelUpdateRequest", input)
			req, err := toVercelUpdateRequest(newRec)
			providers.EndToNative(c.observer, "toVercelUpdateRequest", before, input, req, err)
			if err != nil {
				return err
			}
			_, err = c.UpdateDNSRecord(ctx, existingID, req)
			return err
		},
	}
}

func (c *vercelProvider) mkDeleteCorrection(domain string, oldRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			ctx := context.Background()
			existingID := oldRec.Original.(DNSRecord).ID
			return c.DeleteDNSRecord(ctx, domain, existingID)
		},
	}
}

// toVercelCreateRequest converts a RecordConfig to a Vercel CreateDNSRecordRequest.
func toVercelCreateRequest(domain string, rc *models.RecordConfig) (createDNSRecordRequest, error) {
	req := createDNSRecordRequest{}

	name := rc.GetLabel()
	if name == "@" {
		name = ""
	}
	req.Name = name
	req.Domain = domain
	req.Type = rc.Type
	req.TTL = int64(rc.TTL)
	req.Comment = ""

	switch rc.TypeNum {
	case dnsv2.TypeMX:
		f := rc.AsMX()
		req.MXPriority = int64(f.Preference)
		req.Value = new(f.Mx)
	case dnsv2.TypeSRV:
		f := rc.AsSRV()
		req.SRV = &vercelClient.SRV{
			Priority: int64(f.Priority),
			Weight:   int64(f.Weight),
			Port:     int64(f.Port),
			Target:   f.Target,
		}
		// When dealing with SRV records, we must not set the Value fields,
		// otherwise the API throws an error:
		// bad_request - Invalid request: should NOT have additional property `value`
		req.Value = nil
	case dnsv2.TypeTXT:
		req.Value = new(rc.GetTargetTXTJoined())
	case dnsv2.TypeHTTPS:
		f := rc.AsHTTPS()
		req.HTTPS = &httpsRecord{
			Priority: int64(f.Priority),
			Target:   f.Target,
			Params:   models.Svcbv2ValueToString(rc.AsHTTPS().Value),
		}
		// When dealing with HTTPS records, we must not set the Value fields,
		// otherwise the API throws an error:
		// bad_request - Invalid request: should NOT have additional property `value`.
		req.Value = nil
	case dnsv2.TypeCAA:
		f := rc.AsCAA()
		req.Value = new(fmt.Sprintf(`%v %s "%s"`, f.Flag, f.Tag, f.Value))
	default:
		req.Value = new(rc.GetRDATA().String())
	}

	return req, nil
}

// toVercelUpdateRequest converts a RecordConfig to a Vercel UpdateDNSRecordRequest.
func toVercelUpdateRequest(rc *models.RecordConfig) (updateDNSRecordRequest, error) {
	req := updateDNSRecordRequest{}

	name := rc.GetLabel()
	if name == "@" {
		name = ""
	}
	req.Name = &name

	req.TTL = new(int64(rc.TTL))
	req.Comment = ""

	switch rc.TypeNum {
	case dnsv2.TypeMX:
		f := rc.AsMX()
		req.MXPriority = new(int64(f.Preference))
		req.Value = new(f.Mx)
	case dnsv2.TypeSRV:
		f := rc.AsSRV()
		req.SRV = &vercelClient.SRVUpdate{
			Priority: new(int64(f.Priority)),
			Weight:   new(int64(f.Weight)),
			Port:     new(int64(f.Port)),
			Target:   new(f.Target),
		}
		// When dealing with SRV records, we must not set the Value fields,
		// otherwise the API throws an error:
		// bad_request - Invalid request: should NOT have additional property `value`
		req.Value = nil
	case dnsv2.TypeTXT:
		txtValue := rc.GetTargetTXTJoined()
		req.Value = &txtValue
	case dnsv2.TypeHTTPS:
		f := rc.AsHTTPS()
		req.HTTPS = &httpsRecord{
			Priority: int64(f.Priority),
			Target:   f.Target,
			Params:   models.Svcbv2ValueToString(f.Value),
		}
		// When dealing with HTTPS records, we must not set the Value fields,
		// otherwise the API throws an error:
		// bad_request - Invalid request: should NOT have additional property `value`.
		req.Value = nil
	case dnsv2.TypeCAA:
		f := rc.AsCAA()
		value := fmt.Sprintf(`%v %s "%s"`, f.Flag, f.Tag, f.Value)
		req.Value = &value
	default:
		value := rc.GetRDATA().String()
		req.Value = &value
	}

	return req, nil
}

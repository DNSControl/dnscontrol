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

	"codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
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
	client   vercelClient.Client
	apiToken string
	teamID   string

	createLimiter *rateLimiter
	updateLimiter *rateLimiter
	deleteLimiter *rateLimiter
	listLimiter   *rateLimiter
}

// uint16Zero converts value to uint16 or returns 0, use wisely
//
// Vercel's Go SDK implies int64 for almost everything, but since Vercel doesn't actually
// implement their own NS and instead uses NS1 / Constellix (previously), we'd assume if
// TTL and Priority are int64, they are in fact uint16 and otherwise be rejected by upstream
// providers. Under this assumption, we'd convert int64 to uint16 as wells.
func uint16Zero(value any) uint16 {
	switch v := value.(type) {
	case float64:
		return uint16(v)
	case uint16:
		return v
	case int64:
		return uint16(v)
	case nil:
	}
	return 0
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

	var zoneRecords []*models.RecordConfig

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

		rc := &models.RecordConfig{
			TTL:      uint32(r.TTL),
			Original: r,
		}

		name := r.Name
		if name == "@" {
			name = ""
		}
		rc.SetLabel(name, domain)

		if r.Type == "CNAME" || r.Type == "MX" {
			r.Value = dnsutil.Canonical(r.Value)
		}

		switch rtype := r.RecordType; rtype {
		case "MX":
			if err := rc.SetTargetMX(uint16Zero(r.MXPriority), r.Value); err != nil {
				return nil, fmt.Errorf("unparsable MX record: %w", err)
			}
		case "SRV":
			// Vercel's API doesn't always return SRV as an SRV object.
			// It might return priority in the json field, and the srv as a big string `[weight] [port] [domain]` in json 'value' field.
			// We have to create our own string before passing in.
			// Fallback to parsing from string if SRV object is missing
			// r.Value is "weight port target", we need "priority weight port target"
			if err := rc.PopulateFromString(
				rtype,
				fmt.Sprintf("%d %s", uint16Zero(r.Priority), r.Value),
				domain,
			); err != nil {
				return nil, fmt.Errorf("unparsable SRV record from value: %w", err)
			}
		case "HTTPS":
			// Vercel returns priority in a separate field, and value contains "target params".
			// We need to combine them for PopulateFromString.
			if err := rc.PopulateFromString(
				rtype,
				fmt.Sprintf("%d %s", uint16Zero(r.Priority), r.Value),
				domain,
			); err != nil {
				return nil, fmt.Errorf("unparsable HTTPS record: %w", err)
			}
		case "TXT":
			err := rc.SetTargetTXT(r.Value)
			if err != nil {
				return nil, fmt.Errorf("unparsable TXT record: %w", err)
			}
		default:
			if err := rc.PopulateFromString(rtype, r.Value, domain); err != nil {
				return nil, fmt.Errorf("unparsable record received from vercel: %w", err)
			}
		}

		zoneRecords = append(zoneRecords, rc)
	}

	return zoneRecords, nil
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
			req, err := toVercelCreateRequest(domain, newRec)
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
				req, err := toVercelCreateRequest(domain, newRec)
				if err != nil {
					return err
				}
				_, err = c.CreateDNSRecord(ctx, req)
				return err
			}

			req, err := toVercelUpdateRequest(newRec)
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

	req.Domain = domain

	name := rc.GetLabel()
	if name == "@" {
		name = ""
	}
	req.Name = name
	req.Type = rc.Type
	req.Value = new(rc.GetTargetField())
	req.TTL = int64(rc.TTL)
	req.Comment = ""

	switch rc.Type {
	case "MX":
		req.MXPriority = int64(rc.MxPreference)
	case "SRV":
		req.SRV = &vercelClient.SRV{
			Priority: int64(rc.SrvPriority),
			Weight:   int64(rc.SrvWeight),
			Port:     int64(rc.SrvPort),
			Target:   rc.GetTargetField(),
		}
		// When dealing with SRV records, we must not set the Value fields,
		// otherwise the API throws an error:
		// bad_request - Invalid request: should NOT have additional property `value`
		req.Value = nil
	case "TXT":
		req.Value = new(rc.GetTargetTXTJoined())
	case "HTTPS":
		req.HTTPS = &httpsRecord{
			Priority: int64(rc.SvcPriority),
			Target:   rc.GetTargetField(),
			Params:   rc.SvcParams,
		}
		// When dealing with HTTPS records, we must not set the Value fields,
		// otherwise the API throws an error:
		// bad_request - Invalid request: should NOT have additional property `value`.
		req.Value = nil
	case "CAA":
		req.Value = new(fmt.Sprintf(`%v %s "%s"`, rc.CaaFlag, rc.CaaTag, rc.GetTargetField()))
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

	value := rc.GetTargetField()
	req.Value = &value

	req.TTL = new(int64(rc.TTL))
	req.Comment = ""

	switch rc.Type {
	case "MX":
		req.MXPriority = new(int64(rc.MxPreference))
	case "SRV":
		req.SRV = &vercelClient.SRVUpdate{
			Priority: new(int64(rc.SrvPriority)),
			Weight:   new(int64(rc.SrvWeight)),
			Port:     new(int64(rc.SrvPort)),
			Target:   &value,
		}
		// When dealing with SRV records, we must not set the Value fields,
		// otherwise the API throws an error:
		// bad_request - Invalid request: should NOT have additional property `value`
		req.Value = nil
	case "TXT":
		txtValue := rc.GetTargetTXTJoined()
		req.Value = &txtValue
	case "HTTPS":
		req.HTTPS = &httpsRecord{
			Priority: int64(rc.SvcPriority),
			Target:   rc.GetTargetField(),
			Params:   rc.SvcParams,
		}
		// When dealing with HTTPS records, we must not set the Value fields,
		// otherwise the API throws an error:
		// bad_request - Invalid request: should NOT have additional property `value`.
		req.Value = nil
	case "CAA":
		value := fmt.Sprintf(`%v %s "%s"`, rc.CaaFlag, rc.CaaTag, rc.GetTargetField())
		req.Value = &value
	}

	return req, nil
}

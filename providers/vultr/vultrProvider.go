package vultr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
)

/*

Vultr API DNS provider:

Info required in `creds.json`:
   - token

*/

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "VULTR"
	const providerMaintainer = "@pgaskin"
	fns := providers.DspFuncs{
		Initializer:   NewProvider,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

// vultrProvider represents the Vultr DNSServiceProvider.
type vultrProvider struct {
	client *govultr.Client
}

// defaultNS contains the default nameservers for Vultr.
var defaultNS = []string{
	"ns1.vultr.com",
	"ns2.vultr.com",
}

// NewProvider initializes a Vultr DNSServiceProvider.
func NewProvider(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	token := m["token"]
	if token == "" {
		return nil, errors.New("missing Vultr API token")
	}

	config := &oauth2.Config{}

	client := govultr.NewClient(config.Client(context.Background(), &oauth2.Token{AccessToken: token}))
	client.SetUserAgent("dnscontrol")

	_, _, err := client.Account.Get(context.Background())
	return &vultrProvider{client}, err
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *vultrProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	var curRecords models.Records

	listOptions := &govultr.ListOptions{}
	for {
		records, recordsMeta, _, err := api.client.DomainRecord.List(context.Background(), dc.Name, listOptions)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			r, err := toRecordConfig(dc, record)
			if err != nil {
				return nil, err
			}
			curRecords = append(curRecords, r)
		}
		if recordsMeta.Links.Next == "" {
			break
		}
		listOptions.Cursor = recordsMeta.Links.Next
	}

	return curRecords, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *vultrProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, curRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction

	// Vultr requires all records in a recordset (i.e., a name/type pair) to
	// have the same TTL. The checkRecordSetHasMultipleTTLs validation only
	// warns about this, and most other providers like gcore and huaweicloud
	// also go with the smallest value.
	lowest := map[models.RecordKey]uint32{}
	for _, r := range dc.Records {
		if !r.IsTTLSignificant() {
			continue
		}
		if ttl, ok := lowest[r.Key()]; !ok || r.TTL < ttl {
			lowest[r.Key()] = r.TTL
		}
	}
	for _, r := range dc.Records {
		if ttl, ok := lowest[r.Key()]; ok && r.TTL != ttl {
			printer.Warnf("All TTLs for a rrset (%v) must be the same. Using smaller of %v and %v.\n", r.Key(), r.TTL, ttl)
			r.TTL = ttl
		}
	}

	// Vultr rejects any operation which would result in mixed recordset TTLs,
	// so delete/re-create entire recordsets at a time (this is a bit
	// heavy-handed, but it's simpler than trying to be smart about it, and
	// matches what other provders like akamaiedgedns and alidns do).
	changes, actualChangeCount, err := diff2.ByRecordSet(curRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, change.CreateMessage())
		case diff2.CREATE, diff2.CHANGE, diff2.DELETE:
			oldRecords, newRecords := change.Old, change.New
			corrections = append(corrections, change.CreateCorrection(func() error {
				for _, rc := range oldRecords {
					id := rc.Original.(govultr.DomainRecord).ID
					if err := api.client.DomainRecord.Delete(context.Background(), dc.Name, id); err != nil {
						return err
					}
				}
				for _, rc := range newRecords {
					r := toVultrRecord(rc, "")
					if _, _, err := api.client.DomainRecord.Create(context.Background(), dc.Name, &govultr.DomainRecordCreateReq{
						Name:     r.Name,
						Type:     r.Type,
						Data:     r.Data,
						TTL:      r.TTL,
						Priority: &r.Priority,
					}); err != nil {
						return err
					}
				}
				return nil
			}))
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	dnssecCorrections, dnssecChangeCount, err := api.getDNSSECCorrections(dc)
	if err != nil {
		return nil, 0, err
	}
	corrections = append(corrections, dnssecCorrections...)

	return corrections, actualChangeCount + dnssecChangeCount, nil
}

// getDNSSECCorrections returns corrections that update the domain's DNSSEC state.
func (api *vultrProvider) getDNSSECCorrections(dc *models.DomainConfig) ([]*models.Correction, int, error) {
	if dc.AutoDNSSEC == "" {
		return nil, 0, nil
	}

	domain, _, err := api.client.Domain.Get(context.Background(), dc.Name)
	if err != nil {
		return nil, 0, err
	}
	enabled := domain.DNSSec == "enabled"

	if enabled && dc.AutoDNSSEC == "off" {
		return []*models.Correction{{
			Msg: "Disable DNSSEC",
			F:   func() error { return api.client.Domain.Update(context.Background(), dc.Name, "disabled") },
		}}, 1, nil
	}

	if !enabled && dc.AutoDNSSEC == "on" {
		return []*models.Correction{{
			Msg: "Enable DNSSEC",
			F:   func() error { return api.client.Domain.Update(context.Background(), dc.Name, "enabled") },
		}}, 1, nil
	}

	return nil, 0, nil
}

// GetNameservers gets the Vultr nameservers for a domain.
func (api *vultrProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers(defaultNS)
}

// EnsureZoneExists creates a zone if it does not exist.
func (api *vultrProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	if ok, err := api.isDomainInAccount(domain); err != nil {
		return err
	} else if ok {
		return nil
	}

	// Vultr requires an initial IP, use a dummy one.
	_, _, err := api.client.Domain.Create(context.Background(), &govultr.DomainReq{Domain: domain, IP: "0.0.0.0", DNSSec: "disabled"})
	return err
}

func (api *vultrProvider) isDomainInAccount(domain string) (bool, error) {
	zones, err := api.ListZones()
	if err != nil {
		return false, err
	}
	return slices.Contains(zones, domain), nil
}

// ListZones returns the list of zones (domains) in the account.
func (api *vultrProvider) ListZones() ([]string, error) {
	var zones []string

	listOptions := &govultr.ListOptions{}
	for {
		domains, meta, _, err := api.client.Domain.List(context.Background(), listOptions)
		if err != nil {
			return nil, err
		}
		for _, d := range domains {
			zones = append(zones, d.Domain)
		}
		if meta.Links.Next == "" {
			break
		}
		listOptions.Cursor = meta.Links.Next
	}

	return zones, nil
}

// toRecordConfig converts a Vultr DomainRecord to a RecordConfig. #rtype_variations.
func toRecordConfig(dc *models.DomainConfig, r govultr.DomainRecord) (*models.RecordConfig, error) {
	data := r.Data
	label := dc.LabelFromShort(r.Name)
	ttl := uint32(r.TTL)

	var rc *models.RecordConfig
	var err error
	switch rtype := r.Type; rtype {
	case "CNAME", "NS":
		// Make target into a FQDN if it is a CNAME, NS, MX, or SRV.
		if !strings.HasSuffix(data, ".") {
			data = data + "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, rtype, data)
	case "CAA":
		// Vultr returns CAA records in the format "[flag] [tag] [value]".
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, data)
	case "MX":
		if !strings.HasSuffix(data, ".") {
			data = data + "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Priority, data)
	case "SRV":
		// Vultr returns SRV records in the format "[weight] [port] [target]".
		if !strings.HasSuffix(data, ".") {
			data = data + "."
		}
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, fmt.Sprintf("%d %s", r.Priority, data))
	case "TXT":
		// As of 2026-07-26:
		//	- TXT records from Vultr are always surrounded by quotes.
		//	- Leading/trailing whitespace is removed before any parsing.
		//	- If there is a double-quote at the beginning or the end, but not both, it rejects it (`Record data must be enclosed in quotes`).
		//	- If there are double-quotes at both ends, they are removed.
		//	- If there are any double-quotes left in the string, it rejects it (`Quotes may only appear at the beginning and end of the data`).
		//	- Backslashes within it are always interpreted literally.
		//	- Double-quotes are added to all returned values from the API.
		if !strings.HasPrefix(data, `"`) || !strings.HasSuffix(data, `"`) {
			return nil, errors.New("unexpected lack of quotes in TXT record from Vultr") // break loudly if it changes
		}
		rc, err = dc.NewRecordConfig(label, ttl, rtype, data[1:len(data)-1])

	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, r.Data)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r
	return rc, nil
}

// toVultrRecord converts a RecordConfig converted by toRecordConfig back to a Vultr DomainRecordReq. #rtype_variations.
func toVultrRecord(rc *models.RecordConfig, vultrID string) *govultr.DomainRecord {
	name := rc.GetLabel()
	// Vultr uses a blank string to represent the apex domain.
	if name == "@" {
		name = ""
	}

	data := rc.GetTargetField()

	// Vultr does not use a period suffix for CNAME, NS, MX or SRV.
	data = strings.TrimSuffix(data, ".")

	priority := 0

	if rc.Type == "MX" {
		priority = int(rc.MxPreference)
	}
	if rc.Type == "SRV" {
		priority = int(rc.SrvPriority)
	}

	r := &govultr.DomainRecord{
		ID:       vultrID,
		Type:     rc.Type,
		Name:     name,
		Data:     data,
		TTL:      int(rc.TTL),
		Priority: priority,
	}
	switch rtype := rc.Type; rtype { // #rtype_variations
	case "MX":
		if data == "" {
			// Vultr represents a null MX (RFC 7505) as a literal ".".
			r.Data = "."
		}
	case "SRV":
		if data == "" {
			data = "."
		}
		r.Data = fmt.Sprintf("%v %v %s", rc.SrvWeight, rc.SrvPort, data)
	case "CAA":
		r.Data = fmt.Sprintf(`%v %s "%s"`, rc.CaaFlag, rc.CaaTag, rc.GetTargetField())
	case "SSHFP":
		r.Data = fmt.Sprintf("%d %d %s", rc.SshfpAlgorithm, rc.SshfpFingerprint, rc.GetTargetField())
	case "TXT":
		r.Data = `"` + rc.GetTargetTXTJoined() + `"` // see the toRecordConfig comment
	default:
	}

	return r
}

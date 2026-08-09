package linode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"golang.org/x/oauth2"
)

/*

Linode API DNS provider:

Info required in `creds.json`:
   - token

*/

// Allowed values from the Linode API
// https://www.linode.com/docs/api/domains/#domains-list__responses
var allowedTTLValues = []uint32{
	0,       // Default, currently 1209600 seconds
	300,     // 5 minutes
	3600,    // 1 hour
	7200,    // 2 hours
	14400,   // 4 hours
	28800,   // 8 hours
	57600,   // 16 hours
	86400,   // 1 day
	172800,  // 2 days
	345600,  // 4 days
	604800,  // 1 week
	1209600, // 2 weeks
	2419200, // 4 weeks
}

var srvRegexp = regexp.MustCompile(`^_(?P<Service>\w+)\.\_(?P<Protocol>\w+)$`)

// linodeProvider is the handle for this provider.
type linodeProvider struct {
	client      *http.Client
	baseURL     *url.URL
	domainIndex map[string]int
}

var defaultNameServerNames = []string{
	"ns1.linode.com",
	"ns2.linode.com",
	"ns3.linode.com",
	"ns4.linode.com",
	"ns5.linode.com",
}

// NewLinode creates the provider.
func NewLinode(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	if m["token"] == "" {
		return nil, errors.New("missing Linode token")
	}

	ctx := context.Background()
	client := oauth2.NewClient(
		ctx,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: m["token"]}),
	)

	baseURL, err := url.Parse(defaultBaseURL)
	if err != nil {
		return nil, errors.New("invalid base URL for Linode")
	}

	api := &linodeProvider{client: client, baseURL: baseURL}

	// Get a domain to validate the token
	if err := api.fetchDomainList(); err != nil {
		return nil, err
	}

	return api, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUseCAA:              providers.Can("Linode doesn't support changing the CAA flag"),
	providers.CanUseSRV:              providers.Can("Linode requires non-zero priority"),
	providers.CanUseLOC:              providers.Cannot(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "LINODE"
	const providerMaintainer = "@koesie10"
	// SRV support is in this provider, but Linode doesn't seem to support it properly
	fns := providers.DspFuncs{
		Initializer:   NewLinode,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Linode",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/linode",
		PortalURL:   "https://cloud.linode.com/profile/tokens", // TODO: Verify
		Fields: []providers.CredsField{
			{
				Key:      "token",
				Label:    "API token",
				Help:     "Your Linode Personal Access Token.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

// GetNameservers returns the nameservers for a domain.
func (api *linodeProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers(defaultNameServerNames)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *linodeProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	if api.domainIndex == nil {
		if err := api.fetchDomainList(); err != nil {
			return nil, err
		}
	}
	domainID, ok := api.domainIndex[domain]
	if !ok {
		return nil, fmt.Errorf("'%s' not a zone in Linode account", domain)
	}

	return api.getRecordsForDomain(domainID, dc)
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *linodeProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	// Linode doesn't allow selecting an arbitrary TTL, only a set of predefined values
	// We need to make sure we don't change it every time if it is as close as it's going to get
	// The documentation says that it will always round up to the next highest value: 300 -> 300, 301 -> 3600.
	// https://www.linode.com/docs/api/domains/#domains-list__responses
	for _, record := range dc.Records {
		record.TTL = fixTTL(record.TTL)
	}

	if api.domainIndex == nil {
		if err := api.fetchDomainList(); err != nil {
			return nil, 0, err
		}
	}
	domainID, ok := api.domainIndex[dc.Name]
	if !ok {
		return nil, 0, fmt.Errorf("'%s' not a zone in Linode account", dc.Name)
	}

	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	prefixedCorrections := make(map[int]struct{})
	postfixedCorrections := make(map[int]struct{})

	var corrections []*models.Correction
	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})

		case diff2.CREATE:
			req, err := toReq(dc, change.New[0])
			if err != nil {
				return nil, 0, err
			}
			j, err := json.Marshal(req)
			if err != nil {
				return nil, 0, err
			}
			corrections = append(corrections, &models.Correction{
				Msg: fmt.Sprintf("%s: %s", change.Msgs[0], string(j)),
				F: func() error {
					record, err := api.createRecord(domainID, req)
					if err != nil {
						return err
					}
					// TTL isn't saved when creating a record, so we will need to modify it immediately afterwards
					return api.modifyRecord(domainID, record.ID, req)
				},
			})

		case diff2.DELETE:
			id := change.Old[0].Original.(*domainRecord).ID
			if id == 0 { // Skip ID 0, these are the default nameservers always present
				actualChangeCount--
				continue
			}
			corrections = append(corrections, &models.Correction{
				Msg: fmt.Sprintf("%s, Linode ID: %d", change.Msgs[0], id),
				F: func() error {
					return api.deleteRecord(domainID, id)
				},
			})

		case diff2.CHANGE:
			id := change.Old[0].Original.(*domainRecord).ID
			if id == 0 { // Skip ID 0, these are the default nameservers always present
				actualChangeCount--
				continue
			}
			req, err := toReq(dc, change.New[0])
			if err != nil {
				return nil, 0, err
			}
			j, err := json.Marshal(req)
			if err != nil {
				return nil, 0, err
			}
			corrections = append(corrections, &models.Correction{
				Msg: fmt.Sprintf("%s, Linode ID: %d: %s", change.Msgs[0], id, string(j)),
				F: func() error {
					return api.modifyRecord(domainID, id, req)
				},
			})

		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}

		// Linode is strict about not setting an MX record when a null MX record is present and about not setting a null
		// MX record when an MX record is present. Therefore, we re-sort these specific changes so they always happen
		// first/last.
		if len(change.Old) > 0 && change.Old[0].Type == "MX" && change.Old[0].GetRDATA().String() == "0 ." {
			prefixedCorrections[len(corrections)-1] = struct{}{}
		} else if len(change.New) > 0 && change.New[0].Type == "MX" && change.New[0].GetRDATA().String() == "0 ." {
			postfixedCorrections[len(corrections)-1] = struct{}{}
		}
	}

	sort.SliceStable(corrections, func(i, j int) bool {
		_, iPrefixed := prefixedCorrections[i]
		_, jPrefixed := prefixedCorrections[j]
		if iPrefixed != jPrefixed {
			return iPrefixed
		}
		_, iPostfixed := postfixedCorrections[i]
		_, jPostfixed := postfixedCorrections[j]
		if iPostfixed != jPostfixed {
			return jPostfixed
		}
		return false
	})

	return corrections, actualChangeCount, nil
}

func (api *linodeProvider) getRecordsForDomain(domainID int, dc *models.DomainConfig) (models.Records, error) {
	records, err := api.getRecords(domainID)
	if err != nil {
		return nil, err
	}

	existingRecords := make([]*models.RecordConfig, len(records), len(records)+len(defaultNameServerNames))
	for i := range records {
		existingRecords[i], err = toRc(dc, &records[i])
		if err != nil {
			return nil, err
		}
	}

	// Linode always has read-only NS servers, but these are not mentioned in the API response
	// https://github.com/linode/manager/blob/6503a875f7a4e82dd8335d4ce16fcbd8ae492e21/packages/manager/src/features/Domains/DomainDetail/DomainRecords/DomainRecordsUtils.ts#L59-L125
	for _, name := range defaultNameServerNames {
		rc, err := dc.NewRecordConfig("@", 300, "NS", name+".")
		if err != nil {
			return nil, err
		}
		rc.Original = &domainRecord{}

		existingRecords = append(existingRecords, rc)
	}

	return existingRecords, nil
}

func toRc(dc *models.DomainConfig, r *domainRecord) (*models.RecordConfig, error) {
	label := dc.LabelFromShort(r.Name)
	ttl := r.TTLSec
	var rc *models.RecordConfig
	var err error
	switch rtype := r.Type; rtype { // #rtype_variations
	case "CNAME", "NS":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, strings.TrimSuffix(r.Target, ".")+".")
	case "MX":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Priority, strings.TrimSuffix(r.Target, ".")+".")
	case "SRV":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Priority, r.Weight, r.Port, strings.TrimSuffix(r.Target, ".")+".")
	case "CAA":
		// Linode doesn't support CAA flags and just returns the tag and value separately
		rc, err = dc.NewRecordConfig(label, ttl, rtype, 0, r.Tag, r.Target)
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Target)
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, r.Target)
	}
	if err == nil {
		rc.Original = r
	}
	return rc, err
}

func toReq(dc *models.DomainConfig, rc *models.RecordConfig) (*recordEditRequest, error) {
	req := &recordEditRequest{
		Type: rc.Type,
		Name: rc.GetLabel(),
		TTL:  int(rc.TTL),
	}

	// Linode doesn't use "@", it uses an empty name
	if req.Name == "@" {
		req.Name = ""
	}

	// Linode uses the same property for MX and SRV priority
	switch rc.TypeNum {
	case dnsv2.TypeMX:
		f := rc.AsMX()
		req.Priority = new(int(f.Preference))
		target := fixTarget(f.Mx, dc.Name)
		// Linode doesn't use "." for a null MX record, it uses an empty name
		if target == "." {
			target = ""
		}
		req.Target = target

	case dnsv2.TypeSRV:
		f := rc.AsSRV()
		req.Priority = new(int(f.Priority))
		req.Weight = int(f.Weight)
		req.Port = int(f.Port)

		// From softlayer provider
		// This is to support SRV, it doesn't work yet for Linode
		result := srvRegexp.FindStringSubmatch(req.Name)
		if len(result) != 3 {
			return nil, fmt.Errorf("SRV Record must match format \"_service._protocol\" not %s", req.Name)
		}
		serviceName, protocol := result[1], strings.ToLower(result[2])
		req.Protocol = protocol
		req.Service = serviceName

		req.Name = ""
		req.Target = f.Target

	case dnsv2.TypeTXT:
		req.Target = rc.GetTargetTXTJoined()

	case dnsv2.TypeCNAME:
		f := rc.AsCNAME()
		req.Target = fixTarget(f.Target, dc.Name)

	case dnsv2.TypeCAA:
		f := rc.AsCAA()
		req.Tag = f.Tag
		req.Target = f.Value

	default:
		req.Target = rc.GetRDATA().String()

	}

	return req, nil
}

func fixTarget(target, domain string) string {
	// Linode always wants a fully qualified target name
	if target[len(target)-1] == '.' {
		return target[:len(target)-1]
	}
	return fmt.Sprintf("%s.%s", target, domain)
}

func fixTTL(ttl uint32) uint32 {
	// if the TTL is larger than the largest allowed value, return the largest allowed value
	if ttl > allowedTTLValues[len(allowedTTLValues)-1] {
		return allowedTTLValues[len(allowedTTLValues)-1]
	}

	for _, v := range allowedTTLValues {
		if v >= ttl {
			return v
		}
	}

	return allowedTTLValues[0]
}

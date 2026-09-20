package spaceship

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/namecheap/go-spaceship-sdk/client"
)

const (
	providerName       = "SPACESHIP"
	providerMaintainer = "@rootful"
	apiBaseURL         = "https://spaceship.dev/api/v1"
	minTTL             = 60
	maxTTL             = 3600
	defaultTTL         = 3600
)

var defaultNS = client.DefaultBasicNameserverHosts()

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot("Spaceship does not expose DNSSEC via the public API"),
	providers.CanConcur:              providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseAlias:            providers.Can("Apex ALIAS is stored as CNAME; declare an apex CNAME instead"),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseDSForChildren:    providers.Cannot(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseSVCB:             providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Cannot("The domain must already exist in the Spaceship account"),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	providers.RegisterRegistrarType(providerName, newReg)
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Spaceship",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/spaceship",
		PortalURL:   "https://www.spaceship.com/application/api-manager/",
		Notes:       "Create an API key in Spaceship API Manager with dnsrecords and domains read/write scopes.",
		Fields: []providers.CredsField{
			{
				Key:      "api_key",
				Label:    "API key",
				Help:     "The API key generated in Spaceship API Manager.",
				Secret:   true,
				Required: true,
			},
			{
				Key:      "api_secret",
				Label:    "API secret",
				Help:     "The API secret shown when creating the Spaceship API key.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

type spaceshipProvider struct {
	observer providers.ConversionObserver
	client   *client.Client
	// sleep is time.Sleep, replaced in tests so rate-limit backoff does not
	// cost the wall-clock time it is waiting out.
	sleep func(time.Duration)
}

func (c *spaceshipProvider) SetConversionObserver(observer providers.ConversionObserver) {
	c.observer = observer
}

func newReg(conf map[string]string) (providers.Registrar, error) {
	return newSpaceship(conf)
}

func newDsp(conf map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	return newSpaceship(conf)
}

func newSpaceship(m map[string]string) (*spaceshipProvider, error) {
	apiKey, apiSecret := m["api_key"], m["api_secret"]
	if apiKey == "" || apiSecret == "" {
		return nil, errors.New("missing spaceship api_key or api_secret")
	}
	cl, err := client.NewClient(apiBaseURL, apiKey, apiSecret)
	if err != nil {
		return nil, err
	}
	return &spaceshipProvider{client: cl, sleep: time.Sleep}, nil
}

// GetNameservers returns the nameservers Spaceship uses when it hosts the zone.
func (c *spaceshipProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	info, err := c.getDomainInfo(domain)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(info.Nameservers.Provider, string(client.BasicNameserverProvider)) && len(info.Nameservers.Hosts) >= 2 {
		return models.ToNameservers(info.Nameservers.Hosts)
	}
	return models.ToNameservers(defaultNS)
}

// GetZoneRecords gets the custom-group records of a zone.
// The SDK omits product-group URL redirects and personalNS glue. Those
// records have no documented write API, so DNSControl leaves them alone
// and does not register URL, URL301, or FRAME.
func (c *spaceshipProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	records, err := c.getDNSRecords(dc.Name)
	if err != nil {
		return nil, err
	}

	existing := make(models.Records, 0, len(records))
	for i := range records {
		native := records[i]
		if isApexNS(dc.Name, native) {
			continue
		}
		before := providers.BeginToRC(c.observer, "toRC", native)
		rc, err := toRC(dc, native)
		providers.EndToRC(c.observer, "toRC", before, native, models.Records{rc}, err)
		if err != nil {
			return nil, err
		}
		existing = append(existing, rc)
	}
	return existing, nil
}

// GetZoneRecordsCorrections returns corrections that turn existing records into dc.Records.
func (c *spaceshipProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	checkNSModifications(dc)

	for _, rec := range dc.Records {
		rec.TTL = clampTTL(rec.TTL)
	}

	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, change := range changes {
		var corr *models.Correction
		switch change.Type {
		case diff2.REPORT:
			corr = &models.Correction{Msg: change.MsgsJoined}
		case diff2.CREATE:
			before := providers.BeginToNative(c.observer, "toNative", change.New)
			req, err := toNative(change.New[0])
			providers.EndToNative(c.observer, "toNative", before, change.New, req, err)
			if err != nil {
				return nil, 0, err
			}
			corr = &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					return c.upsertRecords(dc.Name, []client.DNSRecord{req})
				},
			}
		case diff2.CHANGE:
			before := providers.BeginToNative(c.observer, "toNative", change.New)
			req, err := toNative(change.New[0])
			providers.EndToNative(c.observer, "toNative", before, change.New, req, err)
			if err != nil {
				return nil, 0, err
			}
			oldNative, err := nativeFromExisting(change.Old[0])
			if err != nil {
				return nil, 0, err
			}
			if change.HintOnlyTTL {
				corr = &models.Correction{
					Msg: change.Msgs[0],
					F: func() error {
						return c.upsertRecords(dc.Name, []client.DNSRecord{req})
					},
				}
				break
			}
			corr = &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					if err := c.deleteRecords(dc.Name, []client.DNSRecord{oldNative}); err != nil {
						return err
					}
					return c.upsertRecords(dc.Name, []client.DNSRecord{req})
				},
			}
		case diff2.DELETE:
			oldNative, err := nativeFromExisting(change.Old[0])
			if err != nil {
				return nil, 0, err
			}
			corr = &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					return c.deleteRecords(dc.Name, []client.DNSRecord{oldNative})
				},
			}
		default:
			return nil, 0, fmt.Errorf("unhandled change.Type %s", change.Type)
		}
		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}

func nativeFromExisting(rc *models.RecordConfig) (client.DNSRecord, error) {
	if native, ok := rc.Original.(client.DNSRecord); ok {
		return native, nil
	}
	if native, ok := rc.Original.(*client.DNSRecord); ok && native != nil {
		return *native, nil
	}
	return toNative(rc)
}

func clampTTL(ttl uint32) uint32 {
	if ttl == 0 {
		return defaultTTL
	}
	if ttl < minTTL {
		return minTTL
	}
	if ttl > maxTTL {
		return maxTTL
	}
	return ttl
}

// checkNSModifications silently drops apex NS records. The DNS API cannot
// modify apex delegation, and AuditRecords (NsAtApex) already rejects
// user-supplied apex NS before corrections run. Anything reaching here is
// delegation NS synthesized by nameservers.AddNSRecords, so warning would
// only spam every preview/push.
func checkNSModifications(dc *models.DomainConfig) {
	newList := make(models.Records, 0, len(dc.Records))
	for _, rec := range dc.Records {
		if rec.Type == "NS" && rec.GetLabelFQDN() == dc.Name {
			continue
		}
		newList = append(newList, rec)
	}
	dc.Records = newList
}

func isApexNS(domain string, rec client.DNSRecord) bool {
	if !strings.EqualFold(rec.Type, "NS") {
		return false
	}
	name := strings.TrimSuffix(rec.Name, ".")
	return name == "@" || strings.EqualFold(name, domain)
}

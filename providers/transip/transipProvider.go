package transip

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/transip/gotransip/v6"
	"github.com/transip/gotransip/v6/domain"
	"github.com/transip/gotransip/v6/repository"
)

/*

TransIP DNS Provider (transip.nl)

Info required in `creds.json`
	- AccessToken

*/

type transipProvider struct {
	client  *repository.Client
	domains *domain.Repository
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAKAMAICDN:        providers.Cannot(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseAzureAlias:       providers.Cannot(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Cannot(),
	providers.CanUseDNAME:            providers.Cannot(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseDSForChildren:    providers.Cannot(),
	providers.CanUseHTTPS:            providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Can(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseRoute53Alias:     providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseSVCB:             providers.Cannot(),
	providers.CanUseTLSA:             providers.Can(),
	providers.CanUseDNSKEY:           providers.Cannot(),
	providers.DocCreateDomains:       providers.Cannot(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

// NewTransip creates a new TransIP provider.
func NewTransip(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	if m["AccessToken"] == "" && m["PrivateKey"] == "" {
		return nil, errors.New("no TransIP AccessToken or PrivateKey provided")
	}

	if m["PrivateKey"] != "" && m["AccountName"] == "" {
		return nil, errors.New("no AccountName given, required for authenticating with PrivateKey")
	}

	client, err := gotransip.NewClient(gotransip.ClientConfiguration{
		Token:            m["AccessToken"],
		AccountName:      m["AccountName"],
		PrivateKeyReader: strings.NewReader(m["PrivateKey"]),
	})
	if err != nil {
		return nil, fmt.Errorf("TransIP client fail %s", err.Error())
	}

	api := &transipProvider{}
	api.client = &client
	api.domains = &domain.Repository{Client: client}

	return api, nil
}

func init() {
	const providerName = "TRANSIP"
	const providerMaintainer = "@blackshadev"
	fns := providers.DspFuncs{
		Initializer:   NewTransip,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "TransIP",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/transip",
		PortalURL:   "https://www.transip.nl/cp/account/api/",
		Notes:       "TransIP supports two auth methods: a short lived access token, or an account name paired with a long lived private key.",
		Fields: []providers.CredsField{
			{
				Key:      "_authMethod",
				Label:    "Which authentication method do you want to use?",
				Help:     "Access token is quicker; private key lasts longer but needs your account name.",
				Choices:  []string{"Access token", "Account name + private key"},
				Required: true,
				Internal: true,
			},
			{
				Key:      "AccessToken",
				Label:    "Access token",
				Help:     "Personal access token from the TransIP control panel. Has a limited lifetime.",
				Secret:   true,
				Required: true,
				ShowIf:   map[string]string{"_authMethod": "Access token"},
			},
			{
				Key:      "AccountName",
				Label:    "Account name",
				Help:     "Your TransIP account name.",
				Required: true,
				ShowIf:   map[string]string{"_authMethod": "Account name + private key"},
			},
			{
				Key:       "PrivateKey",
				Label:     "Private key (opens $EDITOR)",
				Help:      "Paste the full PEM block including the BEGIN and END lines, then save and close the editor.",
				Multiline: true,
				Required:  true,
				ShowIf:    map[string]string{"_authMethod": "Account name + private key"},
			},
		},
	})
}

func (n *transipProvider) ListZones() ([]string, error) {
	var domains []string

retry:
	domainsMap, err := n.domains.GetAll()
	if retryNeeded(err) {
		goto retry
	}
	if err != nil {
		return nil, err
	}

	for _, domainname := range domainsMap {
		domains = append(domains, domainname.Name)
	}

	sort.Strings(domains)

	return domains, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (n *transipProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, curRecords models.Records) ([]*models.Correction, int, error) {
	removeDomainNameserversFromDomainRecords(dc)

	result, err := diff2.ByZone(curRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	if !result.HasChanges {
		return []*models.Correction{}, result.ActualChangeCount, nil
	}

	msg := fmt.Sprintf("Zone update for %s\n%s", dc.Name, strings.Join(result.Msgs, "\n"))

	corrections := []*models.Correction{
		{
			Msg: msg,
			F: func() error {
				nativeDNSEntries, err := recordsToNative(result.DesiredPlus)
				if err != nil {
					return err
				}

			retry:
				err = n.domains.ReplaceDNSEntries(dc.Name, nativeDNSEntries)
				if retryNeeded(err) {
					goto retry
				}
				return err

			},
		},
	}

	return corrections, result.ActualChangeCount, err
}

// GetZoneRecords returns all records within given zone.
func (n *transipProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domainName := dc.Name

retry:
	entries, err := n.domains.GetDNSEntries(domainName)
	if retryNeeded(err) {
		goto retry
	}
	if err != nil {
		return nil, err
	}

	existingRecords := []*models.RecordConfig{}
	for _, entry := range entries {
		rts, err := nativeToRecord(entry, domainName)
		if err != nil {
			return nil, err
		}
		existingRecords = append(existingRecords, rts)
	}

	return existingRecords, nil
}

// GetNameservers returns the nameservers of the given zone.
func (n *transipProvider) GetNameservers(domainName string) ([]*models.Nameserver, error) {
	var nss []string

retry:
	entries, err := n.domains.GetNameservers(domainName)
	if retryNeeded(err) {
		goto retry
	}
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		nss = append(nss, entry.Hostname)
	}

	return models.ToNameservers(nss)
}

func recordsToNative(records models.Records) ([]domain.DNSEntry, error) {
	entries := make([]domain.DNSEntry, len(records))

	for iX, record := range records {
		entry, err := recordToNative(record)
		if err != nil {
			return nil, err
		}

		entries[iX] = entry
	}

	return entries, nil
}

func recordToNative(config *models.RecordConfig) (domain.DNSEntry, error) {
	return domain.DNSEntry{
		Name:    config.Name,
		Expire:  int(config.TTL),
		Type:    config.Type,
		Content: config.GetTargetCombinedFunc(nil),
	}, nil
}

func nativeToRecord(entry domain.DNSEntry, origin string) (*models.RecordConfig, error) {

	rc := &models.RecordConfig{
		TTL:      uint32(entry.Expire),
		Type:     entry.Type,
		Original: entry,
	}
	rc.SetLabel(entry.Name, origin)
	if err := rc.PopulateFromStringFunc(entry.Type, entry.Content, origin, nil); err != nil {
		return nil, fmt.Errorf("unparsable record received from TransIP: %w", err)
	}

	return rc, nil
}

// removeDomainNameserversFromDomainRecords removes the nameserver records from the dc.Records which are already defined as the Domain nameservers.
func removeDomainNameserversFromDomainRecords(dc *models.DomainConfig) {
	nameserverLookup := map[string]any{}
	for _, nameserver := range dc.Nameservers {
		nameserverLookup[nameserver.Name] = nil
	}

	newList := make([]*models.RecordConfig, 0, len(dc.Records))
	for _, rec := range dc.Records {

		dotLessNameFQDN := strings.TrimRight(rec.GetTargetField(), ".")
		_, recordInDCNameservers := nameserverLookup[dotLessNameFQDN]

		if rec.Type == "NS" && recordInDCNameservers {
			continue
		}

		newList = append(newList, rec)
	}
	dc.Records = newList
}

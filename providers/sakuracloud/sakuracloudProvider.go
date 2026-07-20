package sakuracloud

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

const defaultEndpoint = "https://secure.sakura.ad.jp/cloud/zone/is1a/api/cloud/1.1"

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Cannot(),
	providers.CanUseDNAME:            providers.Cannot(),
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
	providers.CanUseTLSA:             providers.Cannot(),
	providers.CanUseDNSKEY:           providers.Cannot(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "SAKURACLOUD"
	const providerMaintainer = "@ttkzw"
	fns := providers.DspFuncs{
		Initializer:   newSakuracloudDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

type sakuracloudProvider struct {
	api *sakuracloudAPI
}

func newSakuracloudDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newSakuracloud(conf, metadata)
}

// newDSP initializes a Sakura Cloud DNSServiceProvider.
func newSakuracloud(config map[string]string, _ json.RawMessage) (*sakuracloudProvider, error) {
	// config -- the key/values from creds.json
	accessToken := config["access_token"]
	if accessToken == "" {
		return nil, errors.New("access_token is required")
	}

	accessTokenSecret := config["access_token_secret"]
	if accessTokenSecret == "" {
		return nil, errors.New("access_token_secret is required")
	}

	endpoint := config["endpoint"]
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	api, err := newSakuracloudAPI(accessToken, accessTokenSecret, endpoint)
	if err != nil {
		return nil, err
	}
	dsp := &sakuracloudProvider{
		api: api,
	}
	return dsp, nil
}

type errNoExist struct {
	domain string
}

func (e errNoExist) Error() string {
	return fmt.Sprintf("Zone %s not found in your Sakura Cloud account", e.domain)
}

// GetNameservers returns the current nameservers for a domain.
func (s *sakuracloudProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	itemMap, err := s.api.GetCommonServiceItemMap()
	if err != nil {
		return nil, err
	}

	item, ok := itemMap[domain]
	if !ok {
		return nil, errNoExist{domain}
	}

	return models.ToNameservers(item.Status.NS)
}

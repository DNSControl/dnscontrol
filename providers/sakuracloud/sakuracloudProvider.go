package sakuracloud

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

const defaultEndpoint = "https://secure.sakura.ad.jp/cloud/zone/is1a/api/cloud/1.1"

func init() {
	providers.Register[*sakuracloudProvider]("SAKURACLOUD", providers.Definition{
		FriendlyName: "Sakura Cloud",
		Maintainer:   "@ttkzw",
		SupportedTypes: []string{
			"Default",
			"ALIAS",
			"HTTPS",
			"PTR",
			"SVCB",
		},
		CanAutoDNSSEC:          providers.Cannot(),
		CanConcur:              providers.Unimplemented(),
		CanUseDSForChildren:    providers.Cannot(),
		DocDualHost:            providers.Cannot(),
		DocOfficiallySupported: providers.Cannot(),
	})
}

type sakuracloudProvider struct {
	api *sakuracloudAPI
}

// Initialize initializes a fresh provider instance.
func (s *sakuracloudProvider) Initialize(config map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	// config -- the key/values from creds.json
	accessToken := config["access_token"]
	if accessToken == "" {
		return errors.New("access_token is required")
	}

	accessTokenSecret := config["access_token_secret"]
	if accessTokenSecret == "" {
		return errors.New("access_token_secret is required")
	}

	endpoint := config["endpoint"]
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	api, err := newSakuracloudAPI(accessToken, accessTokenSecret, endpoint)
	if err != nil {
		return err
	}
	*s = sakuracloudProvider{
		api: api,
	}
	return nil
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

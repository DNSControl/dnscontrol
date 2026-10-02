package openwrt

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

type openwrtProvider struct {
	auth string
	host string
}

// Initialize initializes a fresh provider instance.
func (c *openwrtProvider) Initialize(conf map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	if conf["username"] == "" {
		return errors.New("missing openwrt username")
	}
	if conf["password"] == "" {
		return errors.New("missing openwrt password")
	}
	if conf["host"] == "" {
		return errors.New("missing openwrt host")
	}

	host := conf["host"]
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	auth, err := getAuthorization(conf["username"], conf["password"], host)
	if err != nil {
		return fmt.Errorf("could not login: %w", err)
	}

	*c = openwrtProvider{auth: auth, host: host}
	return nil
}

func init() {
	providers.Register[*openwrtProvider]("OPENWRT", providers.Definition{
		FriendlyName: "OpenWrt",
		DocsURL:      "https://docs.dnscontrol.org/provider/openwrt",
		Maintainer:   "@huskyistaken",
		Features: providers.DocumentationNotes{
			providers.CanGetZones:            providers.Can(),
			providers.CanUseAlias:            providers.Cannot(),
			providers.CanUseSRV:              providers.Can(),
			providers.DocOfficiallySupported: providers.Cannot(),
		},
	})
}

// GetNameservers returns the nameservers for a domain.
func (c *openwrtProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return []*models.Nameserver{}, nil
}

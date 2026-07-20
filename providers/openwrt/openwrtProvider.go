package openwrt

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

type openwrtProvider struct {
	auth string
	host string
}

func newDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newOpenwrt(conf, metadata)
}

// newOpenwrt creates the provider.
func newOpenwrt(conf map[string]string, _ json.RawMessage) (*openwrtProvider, error) {
	if conf["username"] == "" {
		return nil, errors.New("missing openwrt username")
	}
	if conf["password"] == "" {
		return nil, errors.New("missing openwrt password")
	}
	if conf["host"] == "" {
		return nil, errors.New("missing openwrt host")
	}

	host := conf["host"]
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	auth, err := getAuthorization(conf["username"], conf["password"], host)
	if err != nil {
		return nil, fmt.Errorf("could not login: %w", err)
	}

	return &openwrtProvider{auth: auth, host: host}, nil
}

var features = providers.DocumentationNotes{
	providers.CanGetZones:            providers.Can(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "OPENWRT"
	const providerMaintainer = "@huskyistaken"
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

// GetNameservers returns the nameservers for a domain.
func (c *openwrtProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return []*models.Nameserver{}, nil
}

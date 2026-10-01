package powerdns

import (
	"encoding/json"
	"errors"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	pdns "github.com/mittwald/go-powerdns"
	"github.com/mittwald/go-powerdns/apis/zones"
)

func init() {
	const providerName = "POWERDNS"
	providers.RegisterCustomRecordType("LUA", providerName, "")
	providers.Register[*powerdnsProvider](providerName, providers.Definition{
		FriendlyName: "PowerDNS",
		Maintainer:   "@jpbede",
		Features: providers.DocumentationNotes{
			// The default for unlisted capabilities is 'Cannot'.
			// See providers/capabilities.go for the entire list of capabilities.
			providers.CanAutoDNSSEC:          providers.Can(),
			providers.CanGetZones:            providers.Can(),
			providers.CanConcur:              providers.Unimplemented(),
			providers.CanUseAlias:            providers.Can("Needs to be enabled in PowerDNS first", "https://doc.powerdns.com/authoritative/guides/alias.html"),
			providers.CanUseCAA:              providers.Can(),
			providers.CanUseDS:               providers.Can(),
			providers.CanUseDHCID:            providers.Can(),
			providers.CanUseLOC:              providers.Unimplemented("Normalization within the PowerDNS API seems to be buggy, so disabled", "https://github.com/PowerDNS/pdns/issues/10558"),
			providers.CanUseNAPTR:            providers.Can(),
			providers.CanUseOPENPGPKEY:       providers.Can(),
			providers.CanUsePTR:              providers.Can(),
			providers.CanUseSOA:              providers.Can(),
			providers.CanUseSRV:              providers.Can(),
			providers.CanUseSSHFP:            providers.Can(),
			providers.CanUseTLSA:             providers.Can(),
			providers.CanUseDNAME:            providers.Can("Needs to be enabled in PowerDNS first", "https://doc.powerdns.com/authoritative/settings.html#setting-dname-processing"),
			providers.CanUseHTTPS:            providers.Can(),
			providers.CanUseSVCB:             providers.Can(),
			providers.CanUseDNSKEY:           providers.Can(),
			providers.DocCreateDomains:       providers.Can(),
			providers.DocDualHost:            providers.Can(),
			providers.DocOfficiallySupported: providers.Cannot(),
		},
	})
}

// powerdnsProvider represents the powerdnsProvider DNSServiceProvider.
type powerdnsProvider struct {
	client         pdns.Client
	APIKey         string
	APIUrl         string
	ServerName     string
	DefaultNS      []string             `json:"default_ns"`
	DNSSecOnCreate bool                 `json:"dnssec_on_create"`
	ZoneKind       zones.ZoneKind       `json:"zone_kind"`
	SOAEditAPI     zones.ZoneSOAEditAPI `json:"soa_edit_api,omitempty"`
	UseViews       bool                 `json:"use_views,omitempty"`

	nameservers []*models.Nameserver
}

// Build the variant name for powerdns. this is the domain + "." + the tag
// so dnscontrol "example.com!internal" becomes powerdns "example.com..internal"
// See https://doc.powerdns.com/authoritative/views.html
func (dsp *powerdnsProvider) zoneName(domain string, tag string) string {
	base := canonical(domain)
	if dsp.UseViews && tag != "" {
		return base + "." + tag
	}
	return base
}

// Initialize initializes a fresh provider instance.
func (dsp *powerdnsProvider) Initialize(m map[string]string, metadata json.RawMessage, _ *providers.CreateOptions) error {
	dsp.APIKey = m["apiKey"]
	if dsp.APIKey == "" {
		return errors.New("PowerDNS API Key is required")
	}

	dsp.APIUrl = m["apiUrl"]
	if dsp.APIUrl == "" {
		return errors.New("PowerDNS API URL is required")
	}

	dsp.ServerName = m["serverName"]
	if dsp.ServerName == "" {
		return errors.New("PowerDNS server name is required")
	}

	// load js config
	if len(metadata) != 0 {
		err := json.Unmarshal(metadata, dsp)
		if err != nil {
			return err
		}
	}
	var nss []string
	for _, ns := range dsp.DefaultNS {
		nss = append(nss, ns[0:len(ns)-1])
	}
	var err error
	dsp.nameservers, err = models.ToNameservers(nss)
	if err != nil {
		return err
	}

	var clientErr error
	dsp.client, clientErr = pdns.New(
		pdns.WithBaseURL(dsp.APIUrl),
		pdns.WithAPIKeyAuthentication(dsp.APIKey),
	)
	return clientErr
}

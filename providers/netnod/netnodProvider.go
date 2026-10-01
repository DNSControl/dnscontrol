package netnod

import (
	"encoding/json"
	"errors"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	netnodPrimaryDNS "github.com/netnod/netnod-primary-dns-client"
)

func init() {
	providers.Register[*netnodProvider]("NETNOD", providers.Definition{
		FriendlyName: "Netnod",
		DocsURL:      "https://docs.dnscontrol.org/provider/netnod",
		PortalURL:    "https://www.netnod.se/dns/dns-enterprise-services",
		Notes:        "An API key is required. The API URL defaults to https://primarydnsapi.netnod.se and can be omitted.",
		CredFields: []providers.CredsField{
			{
				Key:      "apiKey",
				Label:    "API key",
				Help:     "API key for the Netnod Primary DNS API.",
				Secret:   true,
				Required: true,
			},
			{
				Key:     "apiUrl",
				Label:   "API URL",
				Help:    "Base URL of the Netnod Primary DNS API. Leave blank to use the default.",
				Default: "https://primarydnsapi.netnod.se",
			},
		},
		Maintainer: "@Netnod @vilhelmprytz",
		Features: providers.DocumentationNotes{
			// The default for unlisted capabilities is 'Cannot'.
			// See providers/capabilities.go for the entire list of capabilities.
			providers.CanAutoDNSSEC:          providers.Cannot(),
			providers.CanGetZones:            providers.Can(),
			providers.CanConcur:              providers.Unimplemented(),
			providers.CanUseAlias:            providers.Can(),
			providers.CanUseCAA:              providers.Can(),
			providers.CanUseDHCID:            providers.Cannot(),
			providers.CanUseDNAME:            providers.Cannot(),
			providers.CanUseDNSKEY:           providers.Cannot(),
			providers.CanUseDS:               providers.Cannot(),
			providers.CanUseDSForChildren:    providers.Can(),
			providers.CanUseHTTPS:            providers.Can(),
			providers.CanUseLOC:              providers.Cannot(),
			providers.CanUseNAPTR:            providers.Can(),
			providers.CanUseOPENPGPKEY:       providers.Cannot(),
			providers.CanUsePTR:              providers.Can(),
			providers.CanUseSOA:              providers.Cannot(),
			providers.CanUseSRV:              providers.Can(),
			providers.CanUseSSHFP:            providers.Can(),
			providers.CanUseSVCB:             providers.Can(),
			providers.CanUseTLSA:             providers.Can(),
			providers.DocCreateDomains:       providers.Can(),
			providers.DocDualHost:            providers.Can(),
			providers.DocOfficiallySupported: providers.Cannot(),
		},
	})
}

// netnodProvider represents the netnodProvider DNSServiceProvider.
type netnodProvider struct {
	client            *netnodPrimaryDNS.Client
	APIKey            string
	APIUrl            string
	DefaultNS         []string `json:"default_ns"`
	AlsoNotify        []string `json:"also_notify"`
	AllowTransferKeys []string `json:"allow_transfer_keys"`

	nameservers []*models.Nameserver
}

// Initialize initializes a fresh provider instance.
func (dsp *netnodProvider) Initialize(m map[string]string, metadata json.RawMessage, _ *providers.CreateOptions) error {
	dsp.APIKey = m["apiKey"]
	if dsp.APIKey == "" {
		return errors.New("netnod API key is required")
	}

	dsp.APIUrl = m["apiUrl"]

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

	dsp.client = netnodPrimaryDNS.NewClient(dsp.APIUrl, dsp.APIKey)
	return nil
}

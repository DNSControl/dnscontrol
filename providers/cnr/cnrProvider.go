package cnr

// Package CNR implements a registrar that uses the CNR api to set name servers. It will self register it's providers when imported.

import (
	"encoding/json"
	"errors"

	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/DNSControl/dnscontrol/v4/pkg/version"
	cnrcl "github.com/centralnicgroup-opensource/rtldev-middleware-go-sdk/v5/apiclient"
)

// Client describes a connection to the CNR API.
type Client struct {
	conf        map[string]string
	APILogin    string
	APIPassword string
	APIEntity   string
	client      *cnrcl.APIClient
}

var features = providers.DocumentationNotes{
	// See providers/capabilities.go for the entire list of capabilities.
	// The default for unlisted capabilities is 'Cannot'.
	// --- Supported Features ---
	providers.CanAutoDNSSEC:          providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot("Actively maintained provider module."),
	// --- Supported record types ---
	// providers.CanUseAKAMAICDN: 	      providers.Cannot(), // can only be supported by Akamai EdgeDns provider
	providers.CanUseAlias: providers.Can("ALIAS records require an unsigned zone served through RCodeZero and cannot be used with DNSSEC-signed zones."),
	// providers.CanUseAzureAlias:		  providers.Cannot(), // can only be supported by Azure provider
	providers.CanUseCAA:           providers.Can(),
	providers.CanUseDHCID:         providers.Can(),
	providers.CanUseDNAME:         providers.Can(),
	providers.CanUseDNSKEY:        providers.Unimplemented("Ask for this feature."),
	providers.CanUseDS:            providers.Unimplemented("Ask for this feature."),
	providers.CanUseDSForChildren: providers.Unimplemented("Ask for this feature."), // CanUseDS implies CanUseDSForChildren
	providers.CanUseHTTPS:         providers.Cannot("Managed via (Query|Add|Modify|Delete)WebFwd API call. Data not accessible via the resource records list. Hard to integrate this into DNSControl by that."),
	providers.CanUseLOC:           providers.Can(),
	providers.CanUseNAPTR:         providers.Can(),
	providers.CanUsePTR:           providers.Can(),
	// providers.CanUseRoute53Alias:	  providers.Cannot(), // can only be supported by AWS Route53 provider
	providers.CanUseSMIMEA: providers.Can(),
	providers.CanUseSOA:    providers.Cannot("The SOA record is managed on the DNSZone directly. Data only accessible via StatusDNSZone Request, not via the resource records list. Hard to integrate this into DNSControl by that."), // supported by bind, honstingde
	providers.CanUseSRV:    providers.Can("SRV records with empty targets are not supported"),
	providers.CanUseSSHFP:  providers.Can(),
	providers.CanUseSVCB:   providers.Can(),
	providers.CanUseTLSA:   providers.Can(),
}

func newProvider(conf map[string]string) (*Client, error) {
	api := &Client{
		conf:   conf,
		client: cnrcl.NewAPIClient(),
	}
	api.client.SetUserAgent("DNSControl", version.Version())
	api.APILogin, api.APIPassword, api.APIEntity = conf["apilogin"], conf["apipassword"], conf["apientity"]
	if conf["debugmode"] == "2" {
		api.client.EnableDebugMode()
	}
	if api.APIEntity != "OTE" && api.APIEntity != "LIVE" {
		return nil, errors.New("wrong api system entity used. use \"OTE\" for OT&E system or \"LIVE\" for Live system")
	}
	if api.APIEntity == "OTE" {
		api.client.UseOTESystem()
	}
	if api.APILogin == "" || api.APIPassword == "" {
		return nil, errors.New("missing login credentials apilogin or apipassword")
	}
	api.client.SetCredentials(api.APILogin, api.APIPassword)
	return api, nil
}

func newReg(conf map[string]string) (providers.Registrar, error) {
	return newProvider(conf)
}

func newDsp(conf map[string]string, meta json.RawMessage) (providers.DNSServiceProvider, error) {
	return newProvider(conf)
}

func init() {
	const providerName = "CNR"
	const providerMaintainer = "@AsifNawaz-cnic"
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterRegistrarType(providerName, newReg)
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "CentralNic Reseller",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/cnr",
		PortalURL:   "https://www.rrpproxy.net/",
		Fields: []providers.CredsField{
			{
				Key:      "apilogin",
				Label:    "API login",
				Help:     "Your CNR API login username.",
				Required: true,
			},
			{
				Key:      "apipassword",
				Label:    "API password",
				Help:     "Your CNR API password.",
				Secret:   true,
				Required: true,
			},
			{
				Key:      "apientity",
				Label:    "API entity",
				Help:     "Use \"OTE\" for the test (OT&E) system or \"LIVE\" for the production system.",
				Choices:  []string{"OTE", "LIVE"},
				Required: true,
			},
			{
				Key:   "debugmode",
				Label: "Debug mode (optional)",
				Help:  "Set to \"2\" to enable verbose API debug logging. Leave blank to disable.",
			},
		},
	})
}

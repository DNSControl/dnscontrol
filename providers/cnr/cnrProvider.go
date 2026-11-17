package cnr

// Package CNR implements a registrar that uses the CNR api to set name servers. It will self register it's providers when imported.

import (
	"encoding/json"
	"errors"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/DNSControl/dnscontrol/v5/pkg/version"
	cnrcl "github.com/centralnicgroup-opensource/rtldev-middleware-go-sdk/v5/apiclient"
)

// Client describes a connection to the CNR API.
type Client struct {
	observer    providers.ConversionObserver
	conf        map[string]string
	APILogin    string
	APIPassword string
	APIEntity   string
	client      *cnrcl.APIClient
}

// SetConversionObserver sets the conversion observer.
func (client *Client) SetConversionObserver(observer providers.ConversionObserver) {
	client.observer = observer
}

// Initialize initializes a fresh provider instance.
func (client *Client) Initialize(conf map[string]string, _ json.RawMessage, options *providers.CreateOptions) error {
	*client = Client{
		conf:   conf,
		client: cnrcl.NewAPIClient(),
	}
	client.client.SetUserAgent("DNSControl", version.Version())
	client.APILogin, client.APIPassword, client.APIEntity = conf["apilogin"], conf["apipassword"], conf["apientity"]
	if conf["debugmode"] == "2" {
		client.client.EnableDebugMode()
	}
	if client.APIEntity != "OTE" && client.APIEntity != "LIVE" {
		return errors.New("wrong api system entity used. use \"OTE\" for OT&E system or \"LIVE\" for Live system")
	}
	if client.APIEntity == "OTE" {
		client.client.UseOTESystem()
	}
	if client.APILogin == "" || client.APIPassword == "" {
		return errors.New("missing login credentials apilogin or apipassword")
	}
	client.client.SetCredentials(client.APILogin, client.APIPassword)
	client.SetConversionObserver(options.WithDefaults().ConversionObserver)
	return nil
}

func init() {
	providers.Register[*Client]("CNR", providers.Definition{
		FriendlyName: "CentralNic Reseller",
		Maintainer:   "@AsifNawaz-cnic",
		SupportedTypes: []string{
			"Basic8",
			"ALIAS:Can:ALIAS records require an unsigned zone served through RCodeZero and cannot be used with DNSSEC-signed zones.",
			"DHCID",
			"DNAME",
			"LOC",
			"NAPTR",
			"PTR",
			"SMIMEA",
			"SSHFP",
			"SVCB",
			"TLSA",
			"DNSKEY:Unimplemented:Ask for this feature.",
			"DS:Unimplemented:Ask for this feature.",
			"HTTPS:Cannot:Managed via (Query|Add|Modify|Delete)WebFwd API call. Data not accessible via the resource records list. Hard to integrate this into DNSControl by that.",
			"SOA:Cannot:The SOA record is managed on the DNSZone directly. Data only accessible via StatusDNSZone Request, not via the resource records list. Hard to integrate this into DNSControl by that.",
			"SRV:Can:SRV records with empty targets are not supported",
		},
		OfficiallySupported: providers.Cannot("Actively maintained provider module."),
		CanAutoDNSSEC:       providers.Can(),
		CanUseDSForChildren: providers.Unimplemented("Ask for this feature."),
		CredFields: []providers.CredsField{
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
				Key:     "debugmode",
				Label:   "Debug mode",
				Help:    "0 turns debug logging off, 1 logs the API commands for each change, 2 also shows the full CNR API communication.",
				Choices: []string{"0", "1", "2"},
				Default: "0",
			},
		},
		PortalURL: "https://www.rrpproxy.net/",
	})
}

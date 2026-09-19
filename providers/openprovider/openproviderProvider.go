package openprovider

import (
	"encoding/json"
	"errors"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

const minimumTTL = uint32(900)

var features = providers.DocumentationNotes{
	providers.CanAutoDNSSEC:          providers.Unimplemented("DNSSEC can be enabled outside DNSControl"),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Cannot(),
	providers.CanUseDNAME:            providers.Cannot(),
	providers.CanUseDNSKEY:           providers.Cannot("DNSSEC keys are managed by OpenProvider"),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseHTTPS:            providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUseOPENPGPKEY:       providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseRP:               providers.Cannot(),
	providers.CanUseSMIMEA:           providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot("The SOA record is managed by OpenProvider"),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseSVCB:             providers.Cannot(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Cannot("OpenProvider manages the authoritative NS records"),
	providers.DocOfficiallySupported: providers.Cannot(),
}

type openproviderProvider struct {
	client *apiClient
}

func init() {
	const providerName = "OPENPROVIDER"
	const providerMaintainer = "@nvanlaerebeke"
	fns := providers.DspFuncs{
		Initializer:   newOpenProvider,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterDefaultTTL(providerName, minimumTTL)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "OpenProvider",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/openprovider",
		PortalURL:   "https://cp.openprovider.eu/",
		Fields: []providers.CredsField{
			{Key: "username", Label: "Username", Required: true},
			{Key: "password", Label: "Password", Required: true, Secret: true},
			{
				Key:     "api_url",
				Label:   "API base URL",
				Help:    "Only needed for a non-production OpenProvider environment.",
				Default: defaultAPIURL,
			},
		},
	})
}

func newOpenProvider(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	username := settings["username"]
	password := settings["password"]
	if username == "" {
		return nil, errors.New("missing OPENPROVIDER username")
	}
	if password == "" {
		return nil, errors.New("missing OPENPROVIDER password")
	}

	apiURL := settings["api_url"]
	if apiURL == "" {
		apiURL = defaultAPIURL
	}
	client, err := newAPIClient(apiURL, username, password)
	if err != nil {
		return nil, err
	}
	return &openproviderProvider{client: client}, nil
}

package inwx

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/nrdcg/goinwx"
	"github.com/pquerna/otp/totp"
	"golang.org/x/net/idna"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

/*
INWX Registrar and DNS provider

Based on this great INWX API implementation:
https://github.com/nrdcg/goinwx

Info required in `creds.json`:
	- username
	- password

Either of the following settings is required when two factor authentication is enabled:
	- totp (TOTP code if 2FA is enabled; best specified as an env variable)
	- totp-key (shared TOTP secret used to generate a valid TOTP code; not recommended since
	            this effectively defeats the purpose of two factor authentication by storing
	            both factors at the same place)

Additional settings available in `creds.json`:
	- sandbox (set to 1 to use the sandbox API from INWX)
*/

// InwxProductionDefaultNs contains the default INWX nameservers.
var InwxProductionDefaultNs = []string{"ns.inwx.de", "ns2.inwx.de", "ns3.inwx.eu"}

// InwxSandboxDefaultNs contains the default INWX nameservers in the sandbox / OTE.
var InwxSandboxDefaultNs = []string{"ns.ote.inwx.de", "ns2.ote.inwx.de"}

// features is used to let dnscontrol know which features are supported by INWX.
var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDS:               providers.Unimplemented("DS records are only supported at the apex and require a different API call that hasn't been implemented yet."),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseLOC:              providers.Unimplemented(),
	providers.CanUseNAPTR:            providers.Can(),
	providers.CanUsePTR:              providers.Can("PTR records with empty targets are not supported"),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

// inwxAPI is a thin wrapper around goinwx.Client.
type inwxAPI struct {
	client      *goinwx.Client
	sandbox     bool
	domainIndex map[string]int // cache of domains existent in the INWX nameserver
}

// init registers the registrar and the domain service provider with dnscontrol.
func init() {
	const providerName = "INWX"
	const providerMaintainer = "@patschi"
	providers.RegisterRegistrarType(providerName, newInwxReg)
	fns := providers.DspFuncs{
		Initializer:   newInwxDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "INWX",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/inwx",
		PortalURL:   "https://www.inwx.com/en/customer",
		Fields: []providers.CredsField{
			{
				Key:      "username",
				Label:    "Username",
				Help:     "Your INWX account username.",
				Required: true,
			},
			{
				Key:      "password",
				Label:    "Password",
				Help:     "Your INWX account password.",
				Secret:   true,
				Required: true,
			},
			{
				Key:   "sandbox",
				Label: "Use sandbox (optional)",
				Help:  "Set to 1 to use the INWX sandbox API (ote.inwx.com) instead of production. Leave blank for production.",
			},
		},
	})
}

// getOTP either returns the TOTPValue or uses TOTPKey and the current time to generate a valid TOTPValue.
func getOTP(TOTPValue string, TOTPKey string) (string, error) {
	if TOTPValue != "" {
		return TOTPValue, nil
	} else if TOTPKey != "" {
		tan, err := totp.GenerateCode(TOTPKey, time.Now())
		if err != nil {
			return "", fmt.Errorf("INWX: Unable to generate TOTP from totp-key: %w", err)
		}
		return tan, nil
	} else {
		return "", errors.New("INWX: two factor authentication required but no TOTP configured")
	}
}

// loginHelper tries to login and then unlocks the account using two factor authentication if required.
func (api *inwxAPI) loginHelper(TOTPValue string, TOTPKey string) error {
	resp, err := api.client.Account.Login()
	if err != nil {
		return errors.New("INWX: Unable to login")
	}

	switch TFA := resp.TFA; TFA {
	case "0":
		if TOTPKey != "" || TOTPValue != "" {
			printer.Printf("INWX: Warning: no TOTP requested by INWX but totp/totp-key is present in `creds.json`\n")
		}
	case "GOOGLE-AUTH":
		tan, err := getOTP(TOTPValue, TOTPKey)
		if err != nil {
			return err
		}

		err = api.client.Account.Unlock(tan)
		if err != nil {
			return fmt.Errorf("INWX: Could not unlock account: %w", err)
		}
	default:
		return fmt.Errorf("INWX: Unknown two factor authentication mode `%s` has been requested", resp.TFA)
	}

	return nil
}

// newInwx initializes inwxAPI and create a session.
func newInwx(m map[string]string) (*inwxAPI, error) {
	username, password := m["username"], m["password"]
	TOTPValue, TOTPKey := m["totp"], m["totp-key"]
	sandbox := m["sandbox"] == "1"

	if username == "" {
		return nil, errors.New("INWX: username must be provided")
	}
	if password == "" {
		return nil, errors.New("INWX: password must be provided")
	}
	if TOTPValue != "" && TOTPKey != "" {
		return nil, errors.New("INWX: totp and totp-key must not be specified at the same time")
	}

	opts := &goinwx.ClientOptions{Sandbox: sandbox}
	client := goinwx.NewClient(username, password, opts)
	api := &inwxAPI{client: client, sandbox: sandbox}

	err := api.loginHelper(TOTPValue, TOTPKey)
	if err != nil {
		return nil, err
	}

	return api, nil
}

// newInwxReg is called to initialize the INWX registrar provider.
func newInwxReg(m map[string]string) (providers.Registrar, error) {
	return newInwx(m)
}

// new InwxDsp is called to initialize the INWX domain service provider.
func newInwxDsp(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newInwx(m)
}

// makeNameserverRecordRequest is a helper function used to convert a RecordConfig to an INWX NS Record Request.
func makeNameserverRecordRequest(domain string, rec *models.RecordConfig) *goinwx.NameserverRecordRequest {
	content := rec.GetTargetField()

	req := &goinwx.NameserverRecordRequest{
		Domain:  domain,
		Type:    rec.Type,
		Content: content,
		Name:    rec.GetLabel(),
		TTL:     int(rec.TTL),
	}

	switch rType := rec.Type; rType {
	/*
	   INWX is a little bit special for CNAME, NS, MX and SRV records:
	   The API will not accept any target with a final dot but will
	   instead always add this final dot internally.
	   Records with empty targets (i.e., records with target ".")
	   are allowed.
	*/
	case "CNAME", "NS", "ALIAS":
		req.Content = content[:len(content)-1]
	case "MX":
		req.Priority = int(rec.MxPreference)
		if content == "." {
			req.Content = content
		} else {
			req.Content = content[:len(content)-1]
		}
	case "SRV":
		req.Priority = int(rec.SrvPriority)
		if content == "." {
			req.Content = fmt.Sprintf("%d %d %v", rec.SrvWeight, rec.SrvPort, content)
		} else {
			req.Content = fmt.Sprintf("%d %d %v", rec.SrvWeight, rec.SrvPort, content[:len(content)-1])
		}
	default:
		req.Content = rec.GetTargetCombined()
	}

	return req
}

// createRecord is used by GetDomainCorrections to create a new record.
func (api *inwxAPI) createRecord(domain string, rec *models.RecordConfig) error {
	req := makeNameserverRecordRequest(domain, rec)
	_, err := api.client.Nameservers.CreateRecord(req)
	return err
}

// updateRecord is used by GetDomainCorrections to update an existing record.
func (api *inwxAPI) updateRecord(RecordID string, rec *models.RecordConfig) error {
	req := makeNameserverRecordRequest("", rec)
	err := api.client.Nameservers.UpdateRecord(RecordID, req)
	return err
}

// deleteRecord is used by GetDomainCorrections to delete a record.
func (api *inwxAPI) deleteRecord(RecordID string) error {
	return api.client.Nameservers.DeleteRecord(RecordID)
}

// isNullMX checks if a record is a null MX record.
func isNullMX(rec *models.RecordConfig) bool {
	return rec.Type == "MX" && rec.MxPreference == 0 && rec.GetTargetField() == "."
}

// AutoDnssecToggle enables and disables AutoDNSSEC for INWX domains.
func (api *inwxAPI) AutoDnssecToggle(dc *models.DomainConfig, corrections []*models.Correction) ([]*models.Correction, error) {

	dnssecStatus, err := api.DNSSecStatus(dc.Name)
	if err != nil {
		return corrections, err
	}

	if dnssecStatus == ManualDNSSECStatus && dc.AutoDNSSEC != "" {
		return corrections, fmt.Errorf("INWX: Domain %s has manual DNSSEC enabled. Disable it before using "+
			"AUTODNSSEC_ON/AUTODNSSEC_OFF", dc.Name)
	}

	if dnssecStatus != AutoDNSSECStatus && dc.AutoDNSSEC == "on" {
		corrections = append(corrections, &models.Correction{
			Msg: color.YellowString("Enable AutoDNSSEC"),
			F: func() error {
				return api.enableAutoDNSSEC(dc.Name)
			},
		})
	}

	if dnssecStatus == AutoDNSSECStatus && dc.AutoDNSSEC == "off" {
		corrections = append(corrections, &models.Correction{
			Msg: color.RedString("Disable AutoDNSSEC"),
			F: func() error {
				return api.disableAutoDNSSEC(dc.Name)
			},
		})
	}
	return corrections, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *inwxAPI) GetZoneRecordsCorrections(dc *models.DomainConfig, foundRecords models.Records) ([]*models.Correction, int, error) {

	// INWX support for Null MX requires special handling.  MX preference changes
	// of >0 to 0 are silently dropped, so if the change includes a null MX record
	// we have to delete then create.  Corrections are compiled separately for
	// deletes, changes and creates and then assembled in that order.
	corrections := []*models.Correction{}
	deletes := []*models.Correction{}
	creates := []*models.Correction{}
	deferred := []*models.Correction{}

	corrections, err := api.AutoDnssecToggle(dc, corrections)
	if err != nil {
		return nil, 0, err
	}

	changes, actualChangeCount, err := diff2.ByRecord(foundRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}
	for _, change := range changes {
		changeMessage := change.MsgsJoined
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: changeMessage})
		case diff2.CHANGE:
			oldRec := change.Old[0]
			newRec := change.New[0]
			if isNullMX(newRec) || isNullMX(oldRec) {
				// changing to or from a Null MX has to be deleted then create
				deletes = append(deletes, &models.Correction{
					Msg: color.RedString("- DELETE %s %s %s ttl=%d", oldRec.GetLabelFQDN(), oldRec.Type,
						oldRec.ToComparableNoTTL(), oldRec.TTL),
					F: func() error {
						return api.deleteRecord(oldRec.Original.(goinwx.NameserverRecord).ID)
					},
				})
				deferred = append(deferred, &models.Correction{
					Msg: color.GreenString("+ CREATE %s %s %s ttl=%d", newRec.GetLabelFQDN(), newRec.Type,
						newRec.ToComparableNoTTL(), newRec.TTL),
					F: func() error {
						return api.createRecord(dc.Name, newRec)
					},
				})
			} else {
				recID := oldRec.Original.(goinwx.NameserverRecord).ID
				corrections = append(corrections, &models.Correction{
					Msg: changeMessage,
					F: func() error {
						return api.updateRecord(recID, newRec)
					},
				})

			}
		case diff2.CREATE:
			creates = append(creates, &models.Correction{
				Msg: changeMessage,
				F: func() error {
					return api.createRecord(dc.Name, change.New[0])
				},
			})
		case diff2.DELETE:
			recID := change.Old[0].Original.(goinwx.NameserverRecord).ID
			deletes = append(deletes, &models.Correction{
				Msg: changeMessage,
				F:   func() error { return api.deleteRecord(recID) },
			})
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}
	corrections = append(deletes, corrections...)
	corrections = append(corrections, creates...)
	corrections = append(corrections, deferred...)
	return corrections, actualChangeCount, nil
}

// getDefaultNameservers returns a string map with default nameservers based on e.g. sandbox mode.
func (api *inwxAPI) getDefaultNameservers() []string {
	if api.sandbox {
		return InwxSandboxDefaultNs
	}
	return InwxProductionDefaultNs
}

// GetNameservers returns the nameservers provisioned for the domain or the
// default INWX nameservers.
func (api *inwxAPI) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers(api.fetchRegistrationNSSet(domain))
}

// GetZoneRecords receives the current records from Inwx and converts them to models.RecordConfig.
func (api *inwxAPI) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	info, err := api.client.Nameservers.Info(&goinwx.NameserverInfoRequest{Domain: domain})
	if err != nil {
		return nil, err
	}

	records := []*models.RecordConfig{}

	for _, record := range info.Records {
		if record.Type == "SOA" {
			continue
		}

		/*
		   INWX is a little bit special for CNAME,NS,MX and SRV records:
		   The API will not accept any target with a final dot but will
		   instead always add this final dot internally.
		   Records with empty targets (i.e. records with target ".")
		   are allowed.
		*/
		rtypeAddDot := map[string]bool{
			"ALIAS": true,
			"CNAME": true,
			"MX":    true,
			"NS":    true,
			"SRV":   true,
			"PTR":   true,
		}
		if rtypeAddDot[record.Type] {
			if record.Type == "MX" && record.Content == "." {
				// null records don't need to be modified
			} else if record.Type == "SRV" && strings.HasSuffix(record.Content, ".") {
				// null targets don't need to be modified
			} else {
				record.Content = record.Content + "."
			}
		}

		rc := &models.RecordConfig{
			TTL:      uint32(record.TTL),
			Original: record,
		}
		rc.SetLabelFromFQDN(record.Name, domain)

		switch rType := record.Type; rType {
		case "MX":
			err = rc.SetTargetMX(uint16(record.Priority), record.Content)
		case "SRV":
			err = rc.SetTargetSRVPriorityString(uint16(record.Priority), record.Content)
		default:
			err = rc.PopulateFromString(rType, record.Content, domain)
		}
		if err != nil {
			return nil, fmt.Errorf("INWX: unparsable record received: %w", err)
		}

		records = append(records, rc)
	}

	return records, nil
}

// ListZones returns the zones configured in INWX.
func (api *inwxAPI) ListZones() ([]string, error) {
	if api.domainIndex == nil { // only pull the data once.
		if err := api.fetchNameserverDomains(); err != nil {
			return nil, err
		}
	}

	var domains []string
	for domain := range api.domainIndex {
		domains = append(domains, domain)
	}

	return domains, nil
}

// updateNameservers is used by GetRegistrarCorrections to update the domain's nameservers.
func (api *inwxAPI) updateNameservers(ns []string, domain string) func() error {
	return func() error {
		request := &goinwx.DomainUpdateRequest{
			Domain:      domain,
			Nameservers: ns,
		}

		_, err := api.client.Domains.Update(request)
		return err
	}
}

// GetRegistrarCorrections is part of the registrar provider and determines if the nameservers have to be updated.
func (api *inwxAPI) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	regNameservers := api.fetchRegistrationNSSet(dc.Name)

	var expected []string
	for _, ns := range dc.Nameservers {
		expected = append(expected, ns.Name)
	}
	sort.Strings(expected)

	foundNameservers := strings.Join(regNameservers, ",")
	expectedNameservers := strings.Join(expected, ",")

	if foundNameservers != expectedNameservers {
		return []*models.Correction{
			{
				Msg: fmt.Sprintf("Update nameservers %s -> %s", foundNameservers, expectedNameservers),
				F:   api.updateNameservers(expected, dc.Name),
			},
		}, nil
	}
	return nil, nil
}

// fetchNameserverDomains returns the domains configured in INWX nameservers.
func (api *inwxAPI) fetchNameserverDomains() error {
	zones := map[string]int{}
	request := &goinwx.NameserverListRequest{}
	page := 1
	for {
		request.Page = page
		info, err := api.client.Nameservers.ListWithParams(request)
		if err != nil {
			return err
		}
		for _, domain := range info.Domains {
			// If this is an IDN domain, Nameservers.List.Domains[].Domain
			// will contain the Unicode name but subsequent calls use the ACE
			// encoded name. We will convert it now for use as the cache key
			aceName, err := idna.ToASCII(domain.Domain)
			if err != nil {
				return err
			}
			zones[aceName] = domain.RoID
		}
		if len(zones) >= info.Count {
			break
		}
		page++
	}
	api.domainIndex = zones
	return nil
}

func (api *inwxAPI) fetchRegistrationNSSet(domain string) []string {
	info, err := api.client.Domains.Info(domain, 0)
	if err != nil {
		return api.getDefaultNameservers()
	}
	sort.Strings(info.Nameservers)
	return info.Nameservers
}

// EnsureZoneExists creates a zone if it does not exist.
func (api *inwxAPI) EnsureZoneExists(domain string, metadata map[string]string) error {
	if api.domainIndex == nil { // only pull the data once.
		if err := api.fetchNameserverDomains(); err != nil {
			return err
		}
	}

	if _, ok := api.domainIndex[domain]; ok {
		return nil // zone exists.
	}

	// creating the zone.
	request := &goinwx.NameserverCreateRequest{
		Domain:      domain,
		Type:        "MASTER",
		Nameservers: api.fetchRegistrationNSSet(domain),
	}
	id, err := api.client.Nameservers.Create(request)
	if err != nil {
		return err
	}
	printer.Printf("Added zone for %s to INWX account with id %d\n", domain, id)
	api.domainIndex[domain] = id
	return nil
}

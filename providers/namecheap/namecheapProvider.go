package namecheap

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	nc "github.com/billputer/go-namecheap"
	"golang.org/x/net/publicsuffix"
)

// NamecheapDefaultNs lists the default nameservers for this provider.
var NamecheapDefaultNs = []string{"dns1.registrar-servers.com", "dns2.registrar-servers.com"}

// namecheapProvider is the handle for this provider.
type namecheapProvider struct {
	APIKEY  string
	APIUser string
	client  *nc.Client
}

const namecheapListZonesPageSize = 100

type domainsGetListResponse struct {
	Status  string                        `xml:"Status,attr"`
	Domains []nc.DomainGetListResult      `xml:"CommandResponse>DomainGetListResult>Domain"`
	Paging  domainsGetListResponsePaging  `xml:"CommandResponse>Paging"`
	Errors  []domainsGetListResponseError `xml:"Errors>Error"`
}

type domainsGetListResponsePaging struct {
	TotalItems  int `xml:"TotalItems"`
	CurrentPage int `xml:"CurrentPage"`
	PageSize    int `xml:"PageSize"`
}

type domainsGetListResponseError struct {
	Number  int    `xml:"Number,attr"`
	Message string `xml:",innerxml"`
}

func (e domainsGetListResponseError) Error() string {
	return fmt.Sprintf("Error %d: %s", e.Number, e.Message)
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur:              providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(), // If you remove this, also update not() statements in integrationTest/integration_test.go
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSRV:              providers.Cannot("The namecheap web console allows you to make SRV records, but their api does not let you read or set them"),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Cannot("Requires domain registered through their service"),
	providers.DocDualHost:            providers.Cannot("Doesn't allow control of apex NS records"),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "NAMECHEAP"
	const providerMaintainer = "@willpower232"
	providers.RegisterRegistrarType(providerName, newReg)
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterCustomRecordType("URL", providerName, "")
	providers.RegisterCustomRecordType("URL301", providerName, "")
	providers.RegisterCustomRecordType("FRAME", providerName, "")
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Namecheap",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/namecheap",
		PortalURL:   "https://ap.www.namecheap.com/settings/tools/apiaccess/",
		Fields: []providers.CredsField{
			{
				Key:      "apiuser",
				Label:    "API user",
				Help:     "Your Namecheap API username (usually your account login).",
				Required: true,
			},
			{
				Key:      "apikey",
				Label:    "API key",
				Help:     "The Namecheap API key generated from the API Access page.",
				Secret:   true,
				Required: true,
			},
			{
				Key:   "BaseURL",
				Label: "Base URL (optional)",
				Help:  "Override the API base URL (for example to use the sandbox). Leave blank to use the production URL.",
			},
		},
	})
}

func newDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newProvider(conf, metadata)
}

func newReg(conf map[string]string) (providers.Registrar, error) {
	return newProvider(conf, nil)
}

func newProvider(m map[string]string, _ json.RawMessage) (*namecheapProvider, error) {
	api := &namecheapProvider{}
	api.APIUser, api.APIKEY = m["apiuser"], m["apikey"]
	if api.APIKEY == "" || api.APIUser == "" {
		return nil, errors.New("missing Namecheap apikey and apiuser")
	}
	api.client = nc.NewClient(api.APIUser, api.APIKEY, api.APIUser)
	// if BaseURL is specified in creds, use that url
	BaseURL, ok := m["BaseURL"]
	if ok {
		api.client.BaseURL = BaseURL
	}
	return api, nil
}

func splitDomain(domain string) (sld string, tld string) {
	tld, _ = publicsuffix.PublicSuffix(domain)
	d, _ := publicsuffix.EffectiveTLDPlusOne(domain)
	sld = strings.Split(d, ".")[0]
	return sld, tld
}

// namecheap has request limiting at unpublished limits
// from support in SEP-2017:
//
//	"The limits for the API calls will be 20/Min, 700/Hour and 8000/Day for one user.
//	 If you can limit the requests within these it should be fine."
//
// this helper performs some api action, checks for rate limited response, and if so, enters a retry loop until it resolves
// if you are consistently hitting this, you may have success asking their support to increase your account's limits.
func doWithRetry(f func() error) {
	// sleep 5 seconds at a time, up to 23 times (1 minute, 15 seconds)
	const maxRetries = 23
	const sleepTime = 5 * time.Second
	var currentRetry int
	for {
		err := f()
		if err == nil {
			return
		}
		if strings.Contains(err.Error(), "unexpected status code from api: 405") {
			currentRetry++
			if currentRetry >= maxRetries {
				return
			}
			printer.Printf("Namecheap rate limit exceeded. Waiting %s to retry.\n", sleepTime)
			time.Sleep(sleepTime)
		} else {
			return
		}
	}
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (n *namecheapProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	sld, tld := splitDomain(domain)
	var records *nc.DomainDNSGetHostsResult
	var err error
	doWithRetry(func() error {
		records, err = n.client.DomainsDNSGetHosts(sld, tld)
		return err
	})
	if err != nil {
		return nil, err
	}

	// namecheap has this really annoying feature where they add some parking records if you have no records.
	// This causes a few problems for our purposes, specifically the integration tests.
	// lets detect that one case and pretend it is a no-op.
	if len(records.Hosts) == 2 {
		if records.Hosts[0].Type == "CNAME" &&
			strings.Contains(records.Hosts[0].Address, "parkingpage") &&
			records.Hosts[1].Type == "URL" {
			// return an empty zone
			return nil, nil
		}
	}

	return toRecords(records, domain)
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (n *namecheapProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, actual models.Records) ([]*models.Correction, int, error) {
	// namecheap does not allow setting @ NS with basic DNS
	dc.Filter(func(r *models.RecordConfig) bool {
		if r.Type == "NS" && r.GetLabel() == "@" {
			if !strings.HasSuffix(r.GetTargetField(), "registrar-servers.com.") {
				printer.Println("\n", r.GetTargetField(), "Namecheap does not support changing apex NS records. Skipping.")
			}
			return false
		}
		return true
	})

	toReport, toCreate, toDelete, toModify, actualChangeCount, err := diff.NewCompat(dc).IncrementalDiff(actual)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	// because namecheap doesn't have selective create, delete, modify,
	// we bundle them all up to send at once.  We *do* want to see the
	// changes though

	var desc []string
	for _, i := range toCreate {
		desc = append(desc, "\n"+i.String())
	}
	for _, i := range toDelete {
		desc = append(desc, "\n"+i.String())
	}
	for _, i := range toModify {
		desc = append(desc, "\n"+i.String())
	}

	// only create corrections if there are changes
	if len(desc) > 0 {
		msg := fmt.Sprintf("GENERATE_ZONE: %s (%d records)%s", dc.Name, len(dc.Records), desc)
		corrections = append(corrections,
			&models.Correction{
				Msg: msg,
				F: func() error {
					return n.generateRecords(dc)
				},
			})
	}

	return corrections, actualChangeCount, nil
}

func toRecords(result *nc.DomainDNSGetHostsResult, origin string) ([]*models.RecordConfig, error) {
	var records []*models.RecordConfig
	for _, dnsHost := range result.Hosts {
		record := models.RecordConfig{
			Type:         dnsHost.Type,
			TTL:          uint32(dnsHost.TTL),
			MxPreference: uint16(dnsHost.MXPref),
			Name:         dnsHost.Name,
		}
		record.SetLabel(dnsHost.Name, origin)

		var err error
		switch dnsHost.Type {
		case "MX":
			err = record.SetTargetMX(uint16(dnsHost.MXPref), dnsHost.Address)
		case "FRAME", "URL", "URL301":
			err = record.SetTarget(dnsHost.Address)
		default:
			err = record.PopulateFromString(dnsHost.Type, dnsHost.Address, origin)
		}
		if err != nil {
			return nil, err
		}

		records = append(records, &record)
	}

	return records, nil
}

func (n *namecheapProvider) generateRecords(dc *models.DomainConfig) error {
	var recs []nc.DomainDNSHost

	id := 1
	for _, r := range dc.Records {
		var value string
		switch rtype := r.Type; rtype { // #rtype_variations
		case "CAA":
			value = r.GetTargetCombined()
		default:
			value = r.GetTargetField()
		}

		rec := nc.DomainDNSHost{
			ID:      id,
			Name:    r.GetLabel(),
			Type:    r.Type,
			Address: value,
			MXPref:  int(r.MxPreference),
			TTL:     int(r.TTL),
		}
		recs = append(recs, rec)
		id++
	}
	sld, tld := splitDomain(dc.Name)
	var err error
	doWithRetry(func() error {
		_, err = n.client.DomainDNSSetHosts(sld, tld, recs)
		return err
	})
	return err
}

// GetNameservers returns the nameservers for a domain.
func (n *namecheapProvider) GetNameservers(domainName string) ([]*models.Nameserver, error) {
	// return default namecheap nameservers
	return models.ToNameservers(NamecheapDefaultNs)
}

func (n *namecheapProvider) ListZones() ([]string, error) {
	var zoneList []string
	page := 1
	for {
		zones, paging, err := n.listZonesPage(page)
		if err != nil {
			return nil, err
		}

		for _, zone := range zones {
			zoneList = append(zoneList, zone.Name)
		}

		if paging.TotalItems == 0 || paging.PageSize == 0 || len(zoneList) >= paging.TotalItems {
			return zoneList, nil
		}

		page++
	}
}

func (n *namecheapProvider) listZonesPage(page int) ([]nc.DomainGetListResult, domainsGetListResponsePaging, error) {
	params := url.Values{}
	params.Set("ApiUser", n.client.ApiUser)
	params.Set("ApiKey", n.client.ApiToken)
	params.Set("UserName", n.client.UserName)
	params.Set("ClientIp", n.client.ClientIp)
	params.Set("Command", "namecheap.domains.getList")
	params.Set("Page", strconv.Itoa(page))
	params.Set("PageSize", strconv.Itoa(namecheapListZonesPageSize))

	encodedParams := params.Encode()
	var err error
	var resp *http.Response
	doWithRetry(func() error {
		req, reqErr := http.NewRequest(http.MethodPost, n.client.BaseURL, strings.NewReader(encodedParams))
		if reqErr != nil {
			err = reqErr
			return reqErr
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Length", strconv.Itoa(len(encodedParams)))
		resp, err = n.client.HttpClient.Do(req)
		return err
	})
	if err != nil {
		return nil, domainsGetListResponsePaging{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, domainsGetListResponsePaging{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, domainsGetListResponsePaging{}, fmt.Errorf("unexpected status code from api: %d", resp.StatusCode)
	}

	parsed := domainsGetListResponse{}
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, domainsGetListResponsePaging{}, err
	}
	if parsed.Status == "ERROR" {
		messages := make([]string, 0, len(parsed.Errors))
		for _, apiErr := range parsed.Errors {
			messages = append(messages, apiErr.Error())
		}
		return nil, domainsGetListResponsePaging{}, errors.New(strings.Join(messages, "\n"))
	}
	if parsed.Status == "" {
		return nil, domainsGetListResponsePaging{}, fmt.Errorf("failed to parse xml from api: %s", bytes.TrimSpace(body))
	}

	return parsed.Domains, parsed.Paging, nil
}

// GetRegistrarCorrections returns corrections to update nameservers.
func (n *namecheapProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	var info *nc.DomainInfo
	var err error
	doWithRetry(func() error {
		info, err = n.client.DomainGetInfo(dc.Name)
		return err
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(info.DNSDetails.Nameservers)
	found := strings.Join(info.DNSDetails.Nameservers, ",")

	desiredNs := []string{}
	for _, d := range dc.Nameservers {
		desiredNs = append(desiredNs, d.Name)
	}
	sort.Strings(desiredNs)
	desired := strings.Join(desiredNs, ",")

	if found != desired {
		parts := strings.SplitN(dc.Name, ".", 2)
		sld, tld := parts[0], parts[1]
		return []*models.Correction{
			{
				Msg: fmt.Sprintf("Change Nameservers from '%s' to '%s'", found, desired),
				F: func() (err error) {
					doWithRetry(func() error {
						_, err = n.client.DomainDNSSetCustom(sld, tld, desired)
						return err
					})
					return
				},
			},
		}, nil
	}
	return nil, nil
}

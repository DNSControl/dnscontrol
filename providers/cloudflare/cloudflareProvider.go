package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/fatih/color"
	"golang.org/x/net/idna"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/DNSControl/dnscontrol/v4/pkg/transform"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/zonecache"
	"github.com/DNSControl/dnscontrol/v4/providers/cloudflare/rtypes/cfsingleredirect"
)

/*

Cloudflare API DNS provider:

Info required in `creds.json`:
   - apikey
   - apiuser
   - accountid (optional)

Record level metadata available:
   - cloudflare_proxy ("on", "off", or "full")

Domain level metadata available:
   - cloudflare_proxy_default ("on", "off", or "full")

 Provider level metadata available:
   - ip_conversions
*/

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAlias:            providers.Can("CF automatically flattens CNAME records into A records dynamically"),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDNSKEY:           providers.Cannot(),
	providers.CanUseDS:               providers.Can(),
	providers.CanUseDSForChildren:    providers.Can(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseLOC:              providers.Can(),
	providers.CanUseNAPTR:            providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Cannot("Cloudflare will not work well in situations where it is not the only DNS server"),
	providers.DocOfficiallySupported: providers.Can(),
}

func init() {
	const providerName = "CLOUDFLAREAPI"
	const providerMaintainer = "@tresni"
	fns := providers.DspFuncs{
		Initializer:   newCloudflare,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterCustomRecordType("CF_WORKER_ROUTE", providerName, "")
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Cloudflare",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/cloudflareapi",
		PortalURL:   "https://dash.cloudflare.com/profile/api-tokens", // TODO: Verify
		Notes:       "Cloudflare supports two auth methods: a scoped API token (recommended) or the legacy global API key paired with the account email.",
		Fields: []providers.CredsField{
			{
				Key:      "_authMethod",
				Label:    "Which authentication method do you want to use?",
				Help:     "API token is scoped and recommended; the global API key requires the account email.",
				Choices:  []string{"API token", "Global API key + email"},
				Required: true,
				Internal: true,
			},
			{
				Key:      "apitoken",
				Label:    "API token",
				Help:     "Cloudflare API token with the required permissions.",
				Secret:   true,
				Required: true,
				ShowIf:   map[string]string{"_authMethod": "API token"},
			},
			{
				Key:      "apiuser",
				Label:    "API user (email)",
				Help:     "Account email used with the global API key.",
				Required: true,
				ShowIf:   map[string]string{"_authMethod": "Global API key + email"},
			},
			{
				Key:      "apikey",
				Label:    "API key",
				Help:     "Cloudflare global API key.",
				Secret:   true,
				Required: true,
				ShowIf:   map[string]string{"_authMethod": "Global API key + email"},
			},
			{
				Key:   "accountid",
				Label: "Account ID (optional)",
				Help:  "Cloudflare account ID. Required to manage zones in a specific account when the credentials have access to more than one.",
			},
		},
	})
}

// cloudflareProvider is the handle for API calls.
type cloudflareProvider struct {
	ipConversions []transform.IPConversion
	ignoredLabels []string
	manageWorkers bool
	accountID     string
	cfClient      *cloudflare.API
	//
	manageSingleRedirects bool // New "Single Redirects"-style redirects.
	//
	// Used by
	tcLogFilename string   // Transcode Log file name
	tcLogFh       *os.File // Transcode Log file handle
	tcZone        string   // Transcode Current zone

	zoneCache zonecache.ZoneCache[cloudflare.Zone]
}

// GetNameservers returns the nameservers for a domain.
func (c *cloudflareProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	z, err := c.zoneCache.GetZone(domain)
	if err != nil {
		return nil, err
	}
	return models.ToNameservers(z.NameServers)
}

// ListZones returns a list of the DNS zones.
func (c *cloudflareProvider) ListZones() ([]string, error) {
	return c.zoneCache.GetZoneNames()
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *cloudflareProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	domainID, err := c.getDomainID(domain)
	if err != nil {
		return nil, err
	}

	type result struct {
		records models.Records
		err     error
	}

	// Prepare channels for concurrent fetching
	mainCh := make(chan result, 1)
	redirectCh := make(chan result, 1)
	workerCh := make(chan result, 1)

	// Fetch DNS records concurrently
	go func() {
		recs, err := c.getRecordsForDomain(domainID, domain)
		mainCh <- result{records: recs, err: err}
	}()

	// Fetch Single Redirects concurrently if enabled
	if c.manageSingleRedirects {
		go func() {
			prs, err := c.getSingleRedirects(dc, domainID)
			redirectCh <- result{records: prs, err: err}
		}()
	} else {
		redirectCh <- result{records: nil, err: nil}
	}

	// Fetch Worker Routes concurrently if enabled
	if c.manageWorkers {
		go func() {
			wrs, err := c.getWorkerRoutes(domainID, domain)
			workerCh <- result{records: wrs, err: err}
		}()
	} else {
		workerCh <- result{records: nil, err: nil}
	}

	// Collect results
	mainRes := <-mainCh
	redirectRes := <-redirectCh
	workerRes := <-workerCh

	if mainRes.err != nil {
		return nil, mainRes.err
	}
	if redirectRes.err != nil {
		return nil, redirectRes.err
	}
	if workerRes.err != nil {
		return nil, workerRes.err
	}

	records := mainRes.records

	for _, rec := range records {
		if rec.TTL == 0 {
			rec.TTL = 1
		}
		// Store the proxy status ("orange cloud") for use by get-zones:
		m := getProxyMetadata(rec)
		if p, ok := m["proxy"]; ok {
			if rec.Metadata == nil {
				rec.Metadata = map[string]string{}
			}
			rec.Metadata["cloudflare_proxy"] = p
		}
	}

	records = append(records, redirectRes.records...)
	records = append(records, workerRes.records...)

	// Normalize
	models.PostProcessRecords(records)

	return records, nil
}

func (c *cloudflareProvider) getDomainID(name string) (string, error) {
	z, err := c.zoneCache.GetZone(name)
	if err != nil {
		return "", err
	}
	return z.ID, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (c *cloudflareProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, records models.Records) ([]*models.Correction, int, error) {
	for _, rec := range dc.Records {
		if rec.Type == "ALIAS" {
			rec.ChangeType("CNAME", dc.Name)
		}
	}

	if err := c.preprocessConfig(dc); err != nil {
		return nil, 0, err
	}

	checkNSModifications(dc)

	domainID, err := c.getDomainID(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	for _, rec := range dc.Records {
		// As per CF-API documentation proxied records are always forced to have a TTL of 1.
		// When not forcing this property change here, dnscontrol tries each time to update
		// the TTL of a record which simply cannot be changed anyway.
		if rec.Metadata[metaProxy] != "off" {
			rec.TTL = 1
		}
	}

	checkNSModifications(dc)

	var corrections []*models.Correction

	// Check if comment/tag management is enabled for this domain
	manageComments := dc.Metadata[metaManageComments] == "true"
	manageTags := dc.Metadata[metaManageTags] == "true"

	// Create a comparable function that includes comments/tags only if management is enabled
	comparableFunc := func(rec *models.RecordConfig) string {
		return genComparableWithMgmt(rec, manageComments, manageTags)
	}

	// Cloudflare is a "ByRecord" API.
	instructions, actualChangeCount, err := diff2.ByRecord(records, dc, comparableFunc)
	if err != nil {
		return nil, 0, err
	}

	for _, inst := range instructions {
		addToFront := false
		var corrs []*models.Correction

		domainID := domainID
		msg := inst.Msgs[0]

		switch inst.Type {
		case diff2.CREATE:
			createRec := inst.New[0]
			corrs = c.mkCreateCorrection(createRec, domainID, msg)
			// DS records must always have a corresponding NS record.
			// Therefore, we create NS records before any DS records.
			addToFront = (createRec.Type == "NS")
		case diff2.CHANGE:
			newrec := inst.New[0]
			oldrec := inst.Old[0]
			corrs = c.mkChangeCorrection(oldrec, newrec, domainID, msg)
		case diff2.DELETE:
			deleteRec := inst.Old[0]
			deleteRecType := deleteRec.Type
			corrs = c.mkDeleteCorrection(deleteRecType, deleteRec, domainID, msg)
			// DS records must always have a corresponding NS record.
			// Therefore, we remove DS records before any NS records.
			addToFront = (deleteRecType == "DS")
		}

		if addToFront {
			corrections = append(corrs, corrections...)
		} else {
			corrections = append(corrections, corrs...)
		}
	}

	// Add universalSSL change when needed
	if changed, newState, err := c.checkUniversalSSL(dc, domainID); err == nil && changed {
		var newStateString string
		if newState {
			newStateString = "enabled"
		} else {
			newStateString = "disabled"
		}
		corrections = append(corrections, &models.Correction{
			Msg: fmt.Sprintf("Universal SSL will be %s for this domain.", newStateString),
			F:   func() error { return c.changeUniversalSSL(domainID, newState) },
		})
	}

	return corrections, actualChangeCount, nil
}

func genComparableWithMgmt(rec *models.RecordConfig, manageComments, manageTags bool) string {
	var parts []string
	if rec.Type == "A" || rec.Type == "AAAA" || rec.Type == "CNAME" {
		proxy := rec.Metadata[metaProxy]
		if proxy != "" {
			if proxy == "on" || proxy == "full" {
				proxy = "true"
			}
			if proxy == "off" {
				proxy = "false"
			}
			parts = append(parts, "proxy="+proxy)
		}
	}
	if rec.Type == "CNAME" {
		flatten := rec.Metadata[metaCNAMEFlatten]
		if flatten == "on" {
			parts = append(parts, "flatten=true")
		} else {
			parts = append(parts, "flatten=false")
		}
	}
	// Include comment in comparison only if comment management is enabled
	if manageComments {
		comment := rec.Metadata[metaComment]
		parts = append(parts, "comment="+comment)
	}
	// Include tags in comparison only if tag management is enabled
	if manageTags {
		tags := rec.Metadata[metaTags]
		parts = append(parts, "tags="+tags)
	}
	return strings.Join(parts, ",")
}

func (c *cloudflareProvider) mkCreateCorrection(newrec *models.RecordConfig, domainID, msg string) []*models.Correction {
	switch newrec.Type {
	case "WORKER_ROUTE":
		return []*models.Correction{{
			Msg: msg,
			F:   func() error { return c.createWorkerRoute(domainID, newrec.GetTargetField()) },
		}}
	case "CLOUDFLAREAPI_SINGLE_REDIRECT":
		return []*models.Correction{{
			Msg: msg,
			F: func() error {
				return c.createSingleRedirect(domainID, *newrec.F.(*cfsingleredirect.SingleRedirectConfig))
			},
		}}
	default:
		return c.createRecDiff2(newrec, domainID, msg)
	}
}

func (c *cloudflareProvider) mkChangeCorrection(oldrec, newrec *models.RecordConfig, domainID string, msg string) []*models.Correction {
	var idTxt string
	switch oldrec.Type {
	case "WORKER_ROUTE":
		idTxt = oldrec.Original.(cloudflare.WorkerRoute).ID
	case "CLOUDFLAREAPI_SINGLE_REDIRECT":
		idTxt = oldrec.F.(*cfsingleredirect.SingleRedirectConfig).SRRRulesetID
	default:
		idTxt = oldrec.Original.(cloudflare.DNSRecord).ID
	}
	msg = msg + color.YellowString(" id=%v", idTxt)

	switch newrec.Type {
	case "CLOUDFLAREAPI_SINGLE_REDIRECT":
		return []*models.Correction{{
			Msg: msg,
			F: func() error {
				return c.updateSingleRedirect(domainID, oldrec, newrec)
			},
		}}
	case "WORKER_ROUTE":
		return []*models.Correction{{
			Msg: msg,
			F: func() error {
				return c.updateWorkerRoute(idTxt, domainID, newrec.GetTargetField())
			},
		}}
	default:
		e := oldrec.Original.(cloudflare.DNSRecord)
		proxy := e.Proxiable && newrec.Metadata[metaProxy] != "off"
		// fmt.Fprintf(os.Stderr, "DEBUG: proxy := %v && %v != off is... %v\n", e.Proxiable, newrec.Metadata[metaProxy], proxy)
		return []*models.Correction{{
			Msg: msg,
			F:   func() error { return c.modifyRecord(domainID, e.ID, proxy, newrec) },
		}}
	}
}

func (c *cloudflareProvider) mkDeleteCorrection(recType string, origRec *models.RecordConfig, domainID string, msg string) []*models.Correction {
	var idTxt string
	switch recType {
	case "PAGE_RULE":
		idTxt = origRec.Original.(cloudflare.PageRule).ID
	case "WORKER_ROUTE":
		idTxt = origRec.Original.(cloudflare.WorkerRoute).ID
	case "CLOUDFLAREAPI_SINGLE_REDIRECT":
		idTxt = origRec.Original.(cloudflare.RulesetRule).ID
	default:
		idTxt = origRec.Original.(cloudflare.DNSRecord).ID
	}
	msg = msg + color.RedString(" id=%v", idTxt)

	correction := &models.Correction{
		Msg: msg,
		F: func() error {
			switch recType {
			// case "PAGE_RULE":
			// 	return c.deletePageRule(origRec.Original.(cloudflare.PageRule).ID, domainID)
			case "WORKER_ROUTE":
				return c.deleteWorkerRoute(origRec.Original.(cloudflare.WorkerRoute).ID, domainID)
			case "CLOUDFLAREAPI_SINGLE_REDIRECT":
				return c.deleteSingleRedirects(domainID, *origRec.F.(*cfsingleredirect.SingleRedirectConfig))
			default:
				return c.deleteDNSRecord(origRec.Original.(cloudflare.DNSRecord), domainID)
			}
		},
	}
	return []*models.Correction{correction}
}

func checkNSModifications(dc *models.DomainConfig) {
	newList := make([]*models.RecordConfig, 0, len(dc.Records))

	punyRoot, err := idna.ToASCII(dc.Name)
	if err != nil {
		punyRoot = dc.Name
	}

	for _, rec := range dc.Records {
		if rec.Type == "NS" && rec.GetLabelFQDN() == punyRoot {
			if strings.HasSuffix(rec.GetTargetField(), ".ns.cloudflare.com.") {
				continue
			}
		}
		newList = append(newList, rec)
	}
	dc.Records = newList
}

func (c *cloudflareProvider) checkUniversalSSL(dc *models.DomainConfig, id string) (changed bool, newState bool, err error) {
	expectedStr := dc.Metadata[metaUniversalSSL]
	if expectedStr == "" {
		return false, false, errors.New("metadata not set")
	}

	if actual, err := c.getUniversalSSL(id); err == nil {
		// convert str to bool
		var expected bool
		if expectedStr == "off" {
			expected = false
		} else {
			expected = true
		}
		// did something change?
		if actual != expected {
			return true, expected, nil
		}
		return false, expected, nil
	}
	return false, false, errors.New("error receiving universal ssl state")
}

const (
	metaProxy          = "cloudflare_proxy"
	metaProxyDefault   = metaProxy + "_default"
	metaOriginalIP     = "original_ip" // TODO(tlim): Unclear what this means.
	metaUniversalSSL   = "cloudflare_universalssl"
	metaCNAMEFlatten   = "cloudflare_cname_flatten"
	metaComment        = "cloudflare_comment"
	metaTags           = "cloudflare_tags"
	metaManageComments = "cloudflare_manage_comments"
	metaManageTags     = "cloudflare_manage_tags"
)

func checkProxyVal(v string) (string, error) {
	v = strings.ToLower(v)
	if v != "on" && v != "off" && v != "full" {
		return "", fmt.Errorf("bad metadata value for cloudflare_proxy: '%s'. Use on/off/full", v)
	}
	return v, nil
}

func checkCNAMEFlattenVal(v string) (string, error) {
	v = strings.ToLower(v)
	if v != "on" && v != "off" {
		return "", fmt.Errorf("bad metadata value for cloudflare_cname_flatten: '%s'. Use on/off", v)
	}
	return v, nil
}

func (c *cloudflareProvider) preprocessConfig(dc *models.DomainConfig) error {

	for _, rec := range dc.Records {
		if rec.Type == "ALIAS" {
			rec.ChangeType("CNAME", dc.Name)
		}
	}

	// Determine the default proxy setting.
	var defProxy string
	var err error
	if defProxy = dc.Metadata[metaProxyDefault]; defProxy == "" {
		defProxy = "off"
	} else {
		defProxy, err = checkProxyVal(defProxy)
		if err != nil {
			return err
		}
	}

	// Check UniversalSSL setting
	if u := dc.Metadata[metaUniversalSSL]; u != "" {
		u = strings.ToLower(u)
		if u != "on" && u != "off" {
			return fmt.Errorf("bad metadata value for %s: '%s'. Use on/off", metaUniversalSSL, u)
		}
	}

	// Normalize the proxy setting for each record.
	// A and CNAMEs: Validate. If null, set to default.
	// else: Make sure it wasn't set.  Set to default.
	// iterate backwards so first defined page rules have highest priority
	for i := len(dc.Records) - 1; i >= 0; i-- {
		rec := dc.Records[i]
		if rec.Metadata == nil {
			rec.Metadata = map[string]string{}
		}
		// cloudflare uses "1" to mean "auto-ttl"
		// if we get here and ttl is not specified
		// use automatic mode instead.
		if rec.TTL == 0 {
			rec.TTL = 1
		}
		if rec.TTL != 1 && rec.TTL < 60 {
			rec.TTL = 60
		}

		if rec.Type != "A" && rec.Type != "CNAME" && rec.Type != "AAAA" && rec.Type != "ALIAS" {
			if rec.Metadata[metaProxy] != "" {
				return fmt.Errorf("cloudflare_proxy set on %v record: %#v cloudflare_proxy=%#v", rec.Type, rec.GetLabel(), rec.Metadata[metaProxy])
			}
			// Force it to off.
			rec.Metadata[metaProxy] = "off"
		} else {
			if val := rec.Metadata[metaProxy]; val == "" {
				rec.Metadata[metaProxy] = defProxy
			} else {
				val, err := checkProxyVal(val)
				if err != nil {
					return err
				}
				rec.Metadata[metaProxy] = val
			}
		}

		// Validate CNAME flattening metadata (only valid on CNAME records)
		if val := rec.Metadata[metaCNAMEFlatten]; val != "" {
			if rec.Type != "CNAME" {
				return fmt.Errorf("cloudflare_cname_flatten set on %v record: %#v (only valid on CNAME records)", rec.Type, rec.GetLabel())
			}
			val, err := checkCNAMEFlattenVal(val)
			if err != nil {
				return err
			}
			rec.Metadata[metaCNAMEFlatten] = val
		}

		// Validate tags (check for reserved cf- prefix)
		if tags := rec.Metadata[metaTags]; tags != "" {
			for tag := range strings.SplitSeq(tags, ",") {
				if strings.HasPrefix(strings.ToLower(tag), "cf-") {
					return fmt.Errorf("cloudflare_tags on %v record %#v contains reserved tag prefix 'cf-': %q", rec.Type, rec.GetLabel(), tag)
				}
			}
		}

		// Validate that CF_COMMENT is only used when CF_MANAGE_COMMENTS is enabled
		if rec.Metadata[metaComment] != "" && dc.Metadata[metaManageComments] != "true" {
			return fmt.Errorf("CF_COMMENT used on %v record %#v but CF_MANAGE_COMMENTS is not enabled for this domain", rec.Type, rec.GetLabel())
		}

		// Validate that CF_TAGS is only used when CF_MANAGE_TAGS is enabled
		if rec.Metadata[metaTags] != "" && dc.Metadata[metaManageTags] != "true" {
			return fmt.Errorf("CF_TAGS used on %v record %#v but CF_MANAGE_TAGS is not enabled for this domain", rec.Type, rec.GetLabel())
		}

		// Ensure metadata keys exist when management is enabled, even if empty.
		// This is needed so that modifyRecord sends an explicit empty value to
		// the API to clear comments/tags, rather than omitting the field (which
		// tells Cloudflare to keep the existing value).
		if dc.Metadata[metaManageComments] == "true" {
			if _, ok := rec.Metadata[metaComment]; !ok {
				rec.Metadata[metaComment] = ""
			}
		}
		if dc.Metadata[metaManageTags] == "true" {
			if _, ok := rec.Metadata[metaTags]; !ok {
				rec.Metadata[metaTags] = ""
			}
		}

		// CNAME flattening and proxy are mutually exclusive (Opinion 6: if ambiguous, forbid it)
		// Cloudflare silently disables flattening when proxy is enabled, which leads to confusing behavior
		if rec.Type == "CNAME" && rec.Metadata[metaCNAMEFlatten] == "on" && rec.Metadata[metaProxy] == "on" {
			return fmt.Errorf("CNAME record %#v has both CF_PROXY_ON and CF_CNAME_FLATTEN_ON set, but these are mutually exclusive; Cloudflare ignores CNAME flattening when proxy is enabled", rec.GetLabel())
		}

		switch rec.Type {
		case "CLOUDFLAREAPI_SINGLE_REDIRECT":
			// SINGLEREDIRECT record types. Verify they are enabled.
			if !c.manageSingleRedirects {
				return errors.New("you must add 'manage_single_redirects: true' metadata to cloudflare provider to use CLOUDFLAREAPI_SINGLE_REDIRECT records")
			}
		case "CF_WORKER_ROUTE":
			// CF_WORKER_ROUTE record types. Encode target as $PATTERN,$SCRIPT
			parts := strings.Split(rec.GetTargetField(), ",")
			if len(parts) != 2 {
				return errors.New("invalid data specified for cloudflare worker record")
			}
			rec.TTL = 1
			rec.Type = "WORKER_ROUTE"
		}
	}

	// look for ip conversions and transform records
	for _, rec := range dc.Records {
		// Only transform A records
		if rec.Type != "A" {
			continue
		}
		// only transform "full"
		if rec.Metadata[metaProxy] != "full" {
			continue
		}
		ip, err := netip.ParseAddr(rec.GetTargetField())
		if err != nil {
			return fmt.Errorf("%s is not a valid ip address", rec.GetTargetField())
		}
		newIP, err := transform.IP(ip, c.ipConversions)
		if err != nil {
			return err
		}
		rec.Metadata[metaOriginalIP] = rec.GetTargetField()
		if err := rec.SetTarget(newIP.String()); err != nil {
			return err
		}
	}

	return nil
}

func (c *cloudflareProvider) LogTranscode(zone string, redirect *cfsingleredirect.SingleRedirectConfig) error {
	// No filename? Don't log anything.
	filename := c.tcLogFilename
	if filename == "" {
		return nil
	}

	// File not opened already? Open it.
	if c.tcLogFh == nil {
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
		if err != nil {
			return err
		}
		c.tcLogFh = f
	}
	fh := c.tcLogFh

	// Output "D(zone)"  if needed.
	var text string
	if c.tcZone != zone {
		text = fmt.Sprintf("D(%q, ...\n", zone)
	}
	c.tcZone = zone

	// Generate the new command and output.
	text = text + fmt.Sprintf("    CF_SINGLE_REDIRECT(%q,\n                       %03d,\n                       '%s',\n                       '%s'\n    ),\n",
		redirect.SRName, redirect.Code,
		redirect.SRWhen, redirect.SRThen)
	_, err := fh.WriteString(text)
	return err
}

func newCloudflare(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	api := &cloudflareProvider{}
	api.zoneCache = zonecache.New(api.fetchAllZones)
	// check api keys from creds json file
	if m["apitoken"] == "" && (m["apikey"] == "" || m["apiuser"] == "") {
		return nil, errors.New("if cloudflare apitoken is not set, apikey and apiuser must be provided")
	}
	if m["apitoken"] != "" && (m["apikey"] != "" || m["apiuser"] != "") {
		return nil, errors.New("if cloudflare apitoken is set, apikey and apiuser should not be provided")
	}

	optRP := cloudflare.UsingRetryPolicy(20, 1, 120)
	// UsingRetryPolicy is documented here:
	// https://pkg.go.dev/github.com/cloudflare/cloudflare-go#UsingRetryPolicy
	// The defaults are UsingRetryPolicy(3, 1, 30)

	var err error
	if m["apitoken"] != "" {
		api.cfClient, err = cloudflare.NewWithAPIToken(m["apitoken"], optRP)
	} else {
		api.cfClient, err = cloudflare.New(m["apikey"], m["apiuser"], optRP)
	}

	if err != nil {
		return nil, fmt.Errorf("cloudflare credentials: %w", err)
	}

	// Check account data if set
	if m["accountid"] != "" {
		api.accountID = m["accountid"]
	}

	debug, err := strconv.ParseBool(os.Getenv("CLOUDFLAREAPI_DEBUG"))
	if err == nil {
		api.cfClient.Debug = debug
	}

	if len(metadata) > 0 {
		parsedMeta := &struct {
			IPConversions string   `json:"ip_conversions"`
			IgnoredLabels []string `json:"ignored_labels"`
			ManageWorkers bool     `json:"manage_workers"`
			//
			ManageSingleRedirects bool   `json:"manage_single_redirects"` // New-style Dynamic "Single Redirects"
			TranscodeLogFilename  string `json:"transcode_log"`           // Log the PAGE_RULE conversions.
		}{}
		err := json.Unmarshal([]byte(metadata), parsedMeta)
		if err != nil {
			return nil, err
		}
		api.manageSingleRedirects = parsedMeta.ManageSingleRedirects
		api.tcLogFilename = parsedMeta.TranscodeLogFilename
		api.manageWorkers = parsedMeta.ManageWorkers
		// ignored_labels:
		api.ignoredLabels = append(api.ignoredLabels, parsedMeta.IgnoredLabels...)
		if len(api.ignoredLabels) > 0 {
			printer.Warnf("Cloudflare 'ignored_labels' configuration is deprecated and might be removed. Please use the IGNORE domain directive to achieve the same effect.\n")
		}
		// parse provider level metadata
		if len(parsedMeta.IPConversions) > 0 {
			api.ipConversions, err = transform.DecodeTransformTable(parsedMeta.IPConversions)
			if err != nil {
				return nil, err
			}
		}
	}
	return api, nil
}

// Used on the "existing" records.
type cfRecData struct {
	Name          string   `json:"name"`
	Target        cfTarget `json:"target"`
	Service       string   `json:"service"`        // SRV
	Proto         string   `json:"proto"`          // SRV
	Priority      uint16   `json:"priority"`       // SRV
	Weight        uint16   `json:"weight"`         // SRV
	Port          uint16   `json:"port"`           // SRV
	Tag           string   `json:"tag"`            // CAA
	Flags         uint16   `json:"flags"`          // CAA/DNSKEY
	Value         string   `json:"value"`          // CAA
	Usage         uint8    `json:"usage"`          // TLSA
	Selector      uint8    `json:"selector"`       // TLSA
	MatchingType  uint8    `json:"matching_type"`  // TLSA
	Certificate   string   `json:"certificate"`    // TLSA
	Algorithm     uint8    `json:"algorithm"`      // SSHFP/DNSKEY/DS
	HashType      uint8    `json:"type"`           // SSHFP
	Fingerprint   string   `json:"fingerprint"`    // SSHFP
	Protocol      uint8    `json:"protocol"`       // DNSKEY
	PublicKey     string   `json:"public_key"`     // DNSKEY
	KeyTag        uint16   `json:"key_tag"`        // DS
	DigestType    uint8    `json:"digest_type"`    // DS
	Digest        string   `json:"digest"`         // DS
	Altitude      float64  `json:"altitude"`       // LOC
	LatDegrees    uint8    `json:"lat_degrees"`    // LOC
	LatDirection  string   `json:"lat_direction"`  // LOC
	LatMinutes    uint8    `json:"lat_minutes"`    // LOC
	LatSeconds    float64  `json:"lat_seconds"`    // LOC
	LongDegrees   uint8    `json:"long_degrees"`   // LOC
	LongDirection string   `json:"long_direction"` // LOC
	LongMinutes   uint8    `json:"long_minutes"`   // LOC
	LongSeconds   float64  `json:"long_seconds"`   // LOC
	PrecisionHorz float64  `json:"precision_horz"` // LOC
	PrecisionVert float64  `json:"precision_vert"` // LOC
	Size          float64  `json:"size"`           // LOC
}

// cfTarget is a SRV target. A null target is represented by an empty string, but
// a dot is so acceptable.
type cfTarget string

// UnmarshalJSON decodes a SRV target from the Cloudflare API. A null target is
// represented by a false boolean or a dot. Domain names are FQDNs without a
// trailing period (as of 2019-11-05).
func (c *cfTarget) UnmarshalJSON(data []byte) error {
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	switch v := obj.(type) {
	case string:
		*c = cfTarget(v)
	case bool:
		if v {
			panic("unknown value for cfTarget bool: true")
		}
		*c = "" // the "." is already added by nativeToRecord
	}
	return nil
}

// MarshalJSON encodes cfTarget for the Cloudflare API. Null targets are
// represented by a single period.
func (c cfTarget) MarshalJSON() ([]byte, error) {
	var obj string
	switch c {
	case "", ".":
		obj = "."
	default:
		obj = string(c)
	}
	return json.Marshal(obj)
}

// DNSControlString returns cfTarget normalized to be a FQDN. Null targets are
// represented by a single period.
func (c cfTarget) FQDN() string {
	return strings.TrimRight(string(c), ".") + "."
}

type cfNaptrRecData struct {
	Flags       string `json:"flags"`
	Order       uint16 `json:"order"`
	Preference  uint16 `json:"preference"`
	Regex       string `json:"regex"`
	Replacement string `json:"replacement"`
	Service     string `json:"service"`
}

// uint16Zero converts value to uint16 or returns 0.
func uint16Zero(value any) uint16 {
	switch v := value.(type) {
	case float64:
		return uint16(v)
	case uint16:
		return v
	case nil:
	}
	return 0
}

// stringDefault returns the value as a string or returns the default value if nil.
func stringDefault(value any, def string) string {
	switch v := value.(type) {
	case string:
		return v
	case nil:
	}
	return def
}

func (c *cloudflareProvider) nativeToRecord(domain string, cr cloudflare.DNSRecord) (*models.RecordConfig, error) {
	// Check for read_only metadata
	// https://github.com/DNSControl/dnscontrol/issues/3850
	if cr.Meta != nil {
		if metaMap, ok := cr.Meta.(map[string]any); ok {
			if readOnly, ok := metaMap["read_only"].(bool); ok && readOnly {
				return nil, nil
			}
		}
	}

	// ALIAS in Cloudflare works like CNAME.
	if cr.Type == "ALIAS" {
		cr.Type = "CNAME"
	}

	// workaround for https://github.com/DNSControl/dnscontrol/issues/446
	if cr.Type == "SPF" {
		cr.Type = "TXT"
	}

	// normalize cname,mx,ns records with dots to be consistent with our config format.
	if cr.Type == "ALIAS" || cr.Type == "CNAME" || cr.Type == "MX" || cr.Type == "NS" || cr.Type == "PTR" {
		if cr.Content != "." {
			cr.Content = cr.Content + "."
		}
	}

	rc := &models.RecordConfig{
		TTL:      uint32(cr.TTL),
		Original: cr,
		Metadata: map[string]string{},
	}
	rc.SetLabelFromFQDN(cr.Name, domain)

	if cr.Type == "A" || cr.Type == "AAAA" || cr.Type == "CNAME" {
		if cr.Proxied != nil {
			if *(cr.Proxied) {
				rc.Metadata[metaProxy] = "on"
			} else {
				rc.Metadata[metaProxy] = "off"
			}
		}
	}

	// Check for CNAME flattening setting
	if cr.Type == "CNAME" {
		if cr.Settings.FlattenCNAME != nil && *cr.Settings.FlattenCNAME {
			rc.Metadata[metaCNAMEFlatten] = "on"
		} else {
			rc.Metadata[metaCNAMEFlatten] = "off"
		}
	}

	// Read comment and tags from API response
	if cr.Comment != "" {
		rc.Metadata[metaComment] = cr.Comment
	}
	if len(cr.Tags) > 0 {
		rc.Metadata[metaTags] = strings.Join(cr.Tags, ",")
	}

	switch rType := cr.Type; rType { // #rtype_variations
	case "MX":
		if err := rc.SetTargetMX(*cr.Priority, cr.Content); err != nil {
			return nil, fmt.Errorf("unparsable MX record received from cloudflare: %w", err)
		}
	case "SRV":
		data := cr.Data.(map[string]any)

		target := stringDefault(data["target"], "MISSING.TARGET")
		if target != "." {
			target += "."
		}
		if err := rc.SetTargetSRV(uint16Zero(data["priority"]), uint16Zero(data["weight"]), uint16Zero(data["port"]),
			target); err != nil {
			return nil, fmt.Errorf("unparsable SRV record received from cloudflare: %w", err)
		}
	case "TXT":
		s, err := parseCfTxtContent(cr.Content)
		if err != nil {
			return rc, err
		}
		err = rc.SetTargetTXT(s)
		return rc, err
	default:
		if err := rc.PopulateFromString(rType, cr.Content, domain); err != nil {
			return nil, fmt.Errorf("unparsable record received from cloudflare: %w", err)
		}
	}

	return rc, nil
}

func parseCfTxtContent(s string) (string, error) {
	// Cloudflare encodes TXT records in a mystery format. They tell you when
	// you've done something wrong, but won't document what they do want.
	// If you use their web dashboard and enter the string as any normal human
	// would, they display a warning that you're a bad person and should feel
	// bad for doing that.  However, they accept it just fine, and present it in
	// their API as a string like any person on this planet would expect.  If
	// you enter the string with quotes, they accept that like a BIND zonefile.

	// There is a difference between what you enter in their web dashboard, how
	// it is rewritten by the UI, and what you get in the JSON. Examples:

	// dashboard: i love dns it is great
	// rewritten: "i love dns it is great"
	// seen json: "i love dns it is great"

	// dashboard: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
	// rewritten: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	// seen json: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

	// dashboard: "i love dns" "it is great"
	// rewritten: "i love dns" "it is great"
	// seen json: "i love dns" "it is great"

	// dashboard: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	// rewritten: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	// seen json: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

	// dashboard: "xxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	// rewritten: "xxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	// seen json: "xxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

	// From this we conclude:
	// If it begins and ends with a quote, use ParseQuoted() to decode it.
	// Otherwise, it is a raw string. They could just fucking tell us that in
	// the documenation, but where's the fun in that?

	if s == "" {
		return "", nil
	}
	if s == `"` {
		return "", errors.New("invalid TXT record content: one double quote")
	}
	if s[0] == '"' && s[len(s)-1] == '"' {
		return txtutil.ParseQuoted(s)
	}
	return s, nil
}

func getProxyMetadata(r *models.RecordConfig) map[string]string {
	if r.Type != "A" && r.Type != "AAAA" && r.Type != "CNAME" {
		return nil
	}
	var proxied bool
	if r.Original != nil {
		proxied = *r.Original.(cloudflare.DNSRecord).Proxied
	} else {
		proxied = r.Metadata[metaProxy] != "off"
	}
	return map[string]string{
		"proxy": strconv.FormatBool(proxied),
	}
}

// EnsureZoneExists creates a zone if it does not exist.
func (c *cloudflareProvider) EnsureZoneExists(domain string, metadata map[string]string) error {
	if ok, err := c.zoneCache.HasZone(domain); err != nil || ok {
		return err
	}
	id, err := c.createZone(domain)
	if err != nil {
		return err
	}
	printer.Printf("Added zone for %s to Cloudflare account: %s\n", domain, id)
	return nil
}

// PrepareCloudflareTestWorkers creates Cloudflare Workers required for CF_WORKER_ROUTE integration tests.
func PrepareCloudflareTestWorkers(prv providers.DNSServiceProvider) error {
	cf, ok := prv.(*cloudflareProvider)
	if ok {
		err := cf.createTestWorker("dnscontrol_integrationtest_cnn")
		if err != nil {
			return err
		}

		err = cf.createTestWorker("dnscontrol_integrationtest_msnbc")
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *cloudflareProvider) createTestWorker(workerName string) error {
	wp := cloudflare.CreateWorkerParams{
		ScriptName: workerName,
		Script: `
			addEventListener("fetch", (event) => {
				event.respondWith(
					new Response("Ok.", { status: 200 })
				);
			});`,
	}

	_, err := c.cfClient.UploadWorker(context.Background(), cloudflare.AccountIdentifier(c.accountID), wp)
	return err
}

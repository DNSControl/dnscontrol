package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
	"github.com/cloudflare/cloudflare-go"
	"golang.org/x/net/idna"
)

func (c *cloudflareProvider) fetchAllZones() (map[string]cloudflare.Zone, error) {
	zones, err := c.cfClient.ListZones(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed fetching domain list from cloudflare(%q): %w", c.cfClient.APIEmail, err)
	}

	m := make(map[string]cloudflare.Zone, len(zones))
	for _, zone := range zones {
		if encoded, err := idna.ToASCII(zone.Name); err == nil && encoded != zone.Name {
			if _, ok := m[encoded]; ok {
				fmt.Printf("WARNING: Zone %q appears twice in this cloudflare account\n", encoded)
			}
			m[encoded] = zone
		}
		if _, ok := m[zone.Name]; ok {
			fmt.Printf("WARNING: Zone %q appears twice in this cloudflare account\n", zone.Name)
		}
		m[zone.Name] = zone
	}
	return m, nil
}

// get all records for a domain.
func (c *cloudflareProvider) getRecordsForDomain(id string, dc *models.DomainConfig) ([]*models.RecordConfig, error) {
	var records models.Records
	rrs, _, err := c.cfClient.ListDNSRecords(context.Background(), cloudflare.ZoneIdentifier(id), cloudflare.ListDNSRecordsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed fetching record list from cloudflare(%q): %w", c.cfClient.APIEmail, err)
	}
	for _, rec := range rrs {
		rt, err := c.nativeToRecord(dc, rec)
		if err != nil {
			return nil, err
		}
		// nativeToRecord may return nil if the record is supposed to be skipped
		// i.e. read only, cloudflare-managed, etc.
		if rt != nil {
			records = append(records, rt)
		}
	}
	return records, nil
}

func (c *cloudflareProvider) deleteDNSRecord(rec cloudflare.DNSRecord, domainID string) error {
	return c.cfClient.DeleteDNSRecord(context.Background(), cloudflare.ZoneIdentifier(domainID), rec.ID)
}

func (c *cloudflareProvider) createZone(domainName string) (string, error) {
	zone, err := c.cfClient.CreateZone(context.Background(), domainName, false, cloudflare.Account{ID: c.accountID}, "full")
	if err != nil {
		return "", err
	}
	if encoded, err := idna.ToASCII(zone.Name); err == nil && encoded != zone.Name {
		c.zoneCache.SetZone(encoded, zone)
	}
	c.zoneCache.SetZone(domainName, zone)
	return zone.ID, nil
}

func cfDnskeyData(rec *models.RecordConfig) *cfRecData {
	f := rec.AsDNSKEY()
	return &cfRecData{
		Algorithm: f.Algorithm,
		Flags:     f.Flags,
		Protocol:  f.Protocol,
		PublicKey: f.PublicKey,
	}
}

func cfDSData(rec *models.RecordConfig) *cfRecData {
	f := rec.AsDS()
	return &cfRecData{
		KeyTag:     f.KeyTag,
		Algorithm:  f.Algorithm,
		DigestType: f.DigestType,
		Digest:     f.Digest,
	}
}

func cfSrvData(rec *models.RecordConfig) *cfRecData {
	f := rec.AsSRV()
	serverParts := strings.Split(rec.GetLabelFQDN(), ".")
	c := &cfRecData{
		Service:  serverParts[0],
		Proto:    serverParts[1],
		Name:     strings.Join(serverParts[2:], "."),
		Port:     f.Port,
		Priority: f.Priority,
		Weight:   f.Weight,
	}
	c.Target = cfTarget(f.Target)
	return c
}

func cfCaaData(rec *models.RecordConfig) *cfRecData {
	f := rec.AsCAA()
	return &cfRecData{
		Tag:   f.Tag,
		Flags: uint16(f.Flag),
		Value: f.Value,
	}
}

func cfTlsaData(rec *models.RecordConfig) *cfRecData {
	f := rec.AsTLSA()
	return &cfRecData{
		Usage:        f.Usage,
		Selector:     f.Selector,
		MatchingType: f.MatchingType,
		Certificate:  f.Certificate,
	}
}

func cfSshfpData(rec *models.RecordConfig) *cfRecData {
	f := rec.AsSSHFP()
	return &cfRecData{
		Algorithm:   f.Algorithm,
		HashType:    f.Type,
		Fingerprint: f.FingerPrint,
	}
}

func cfSvcbData(rec *models.RecordConfig) *cfRecData {
	f := rec.AsSVCB()
	return &cfRecData{
		Priority: f.Priority,
		Target:   cfTarget(f.Target),
		Value:    models.Svcbv2ValueToString(f.Value),
	}
}

func cfLocData(rec *models.RecordConfig) *cfRecData {
	f := rec.AsLOC()
	latDir, latDeg, latMin, latSec := models.ReverseLatitude(f.Latitude)
	longDir, longDeg, longMin, longSec := models.ReverseLongitude(f.Longitude)

	return &cfRecData{
		Altitude:      models.ReverseAltitude(f.Altitude),
		LatDegrees:    latDeg,
		LatDirection:  latDir,
		LatMinutes:    latMin,
		LatSeconds:    latSec,
		LongDegrees:   longDeg,
		LongDirection: longDir,
		LongMinutes:   longMin,
		LongSeconds:   longSec,
		PrecisionHorz: models.ReverseENotationInt(f.HorizPre),
		PrecisionVert: models.ReverseENotationInt(f.VertPre),
		Size:          models.ReverseENotationInt(f.Size),
	}
}

func cfNaptrData(rec *models.RecordConfig) *cfNaptrRecData {
	f := rec.AsNAPTR()
	return &cfNaptrRecData{
		Flags:       f.Flags,
		Order:       f.Order,
		Preference:  f.Preference,
		Regex:       f.Regexp,
		Replacement: f.Replacement,
		Service:     f.Service,
	}
}

func (c *cloudflareProvider) createRecDiff2(rec *models.RecordConfig, domainID string, msg string) []*models.Correction {
	var content string
	prio := ""
	priorityNum := uint16(0)
	switch rec.Type {
	case "MX":
		f := rec.AsMX()
		priorityNum = f.Preference
		prio = fmt.Sprintf(" %d ", priorityNum)
		content = f.Mx
	case "TXT":
		content = rec.GetRDATA().String()
	case "DS":
		content = rec.GetRDATA().String()
	default:
		content = rec.GetRDATA().String()
	}
	if rec.Metadata[metaOriginalIP] != "" {
		content = rec.Metadata[metaOriginalIP]
	}
	if msg == "" {
		msg = fmt.Sprintf("CREATE record: %s %s %d%s %s", rec.GetLabel(), rec.Type, rec.TTL, prio, content)
	}
	if rec.Metadata[metaProxy] == "on" || rec.Metadata[metaProxy] == "full" {
		msg = msg + fmt.Sprintf("\nACTIVATE PROXY for new record %s %s %d %s", rec.GetLabel(), rec.Type, rec.TTL, rec.GetRDATA().String())
	}
	if rec.Metadata[metaCNAMEFlatten] == "on" {
		msg = msg + fmt.Sprintf("\nENABLE CNAME FLATTENING for new record %s %s", rec.GetLabel(), rec.Type)
	}
	if rec.Metadata[metaComment] != "" {
		msg = msg + fmt.Sprintf("\nSET COMMENT for new record %s %s: %q", rec.GetLabel(), rec.Type, rec.Metadata[metaComment])
	}
	if rec.Metadata[metaTags] != "" {
		msg = msg + fmt.Sprintf("\nSET TAGS for new record %s %s: %s", rec.GetLabel(), rec.Type, rec.Metadata[metaTags])
	}
	arr := []*models.Correction{{
		Msg: msg,
		F: func() error {
			cf := cloudflare.CreateDNSRecordParams{
				Name:     rec.GetLabel(),
				Type:     rec.Type,
				TTL:      int(rec.TTL),
				Content:  content,
				Priority: &priorityNum,
			}
			// Set comment if specified
			if comment := rec.Metadata[metaComment]; comment != "" {
				cf.Comment = comment
			}
			// Set tags if specified
			if tags := rec.Metadata[metaTags]; tags != "" {
				cf.Tags = strings.Split(tags, ",")
			}
			// Set CNAME flattening setting if enabled
			if rec.Type == "CNAME" && rec.Metadata[metaCNAMEFlatten] == "on" {
				flatten := true
				cf.Settings = cloudflare.DNSRecordSettings{FlattenCNAME: &flatten}
			}
			switch rec.Type {
			case "SRV":
				cf.Data = cfSrvData(rec)
				cf.Name = rec.GetLabelFQDN()
			case "CAA":
				cf.Data = cfCaaData(rec)
				cf.Name = rec.GetLabelFQDN()
				cf.Content = ""
			case "TLSA":
				cf.Data = cfTlsaData(rec)
				cf.Name = rec.GetLabelFQDN()
			case "SSHFP":
				cf.Data = cfSshfpData(rec)
				cf.Name = rec.GetLabelFQDN()
			case "DNSKEY":
				cf.Data = cfDnskeyData(rec)
			case "DS":
				cf.Data = cfDSData(rec)
			case "NAPTR":
				cf.Data = cfNaptrData(rec)
				cf.Name = rec.GetLabelFQDN()
			case "HTTPS", "SVCB":
				cf.Data = cfSvcbData(rec)
			case "LOC":
				cf.Data = cfLocData(rec)
			}
			resp, err := c.cfClient.CreateDNSRecord(context.Background(), cloudflare.ZoneIdentifier(domainID), cf)
			if err != nil {
				return err
			}
			// Records are created with the proxy off. If proxy should be
			// enabled, we do a second API call.
			resultID := resp.ID
			if rec.Metadata[metaProxy] == "on" || rec.Metadata[metaProxy] == "full" {
				return c.modifyRecord(domainID, resultID, true, rec)
			}
			return nil
		},
	}}
	return arr
}

func (c *cloudflareProvider) modifyRecord(domainID, recID string, proxied bool, rec *models.RecordConfig) error {
	if domainID == "" || recID == "" {
		return errors.New("cannot modify record if domain or record id are empty")
	}

	r := cloudflare.UpdateDNSRecordParams{
		ID:      recID,
		Proxied: new(proxied),
		Name:    rec.GetLabel(),
		Type:    rec.Type,
		TTL:     int(rec.TTL),
	}

	// Set comment if specified (nil keeps current, "" empties it, value sets it)
	if comment, ok := rec.Metadata[metaComment]; ok {
		r.Comment = &comment
	}
	// Set tags if specified (empty key means clear all tags)
	if tags, ok := rec.Metadata[metaTags]; ok {
		if tags != "" {
			r.Tags = strings.Split(tags, ",")
		} else {
			r.Tags = []string{}
		}
	}

	switch rec.Type {
	case "TXT":
		r.Content = rec.GetRDATA().String()
	case "MX":
		f := rec.AsMX()
		r.Priority = new(f.Preference)
		r.Content = f.Mx
	case "CNAME":
		// Handle CNAME flattening setting
		flatten := rec.Metadata[metaCNAMEFlatten] == "on"
		r.Settings = cloudflare.DNSRecordSettings{FlattenCNAME: &flatten}
		r.Content = rec.AsCNAME().Target
	case "SRV":
		r.Data = cfSrvData(rec)
		r.Name = rec.GetLabelFQDN()
		r.Content = rec.GetRDATA().String()
	case "CAA":
		r.Data = cfCaaData(rec)
		r.Name = rec.GetLabelFQDN()
		r.Content = ""
	case "TLSA":
		r.Data = cfTlsaData(rec)
		r.Name = rec.GetLabelFQDN()
		r.Content = rec.GetRDATA().String()
	case "SSHFP":
		r.Data = cfSshfpData(rec)
		r.Name = rec.GetLabelFQDN()
		r.Content = rec.GetRDATA().String()
	case "DNSKEY":
		r.Data = cfDnskeyData(rec)
		r.Content = ""
	case "DS":
		r.Data = cfDSData(rec)
		r.Content = ""
	case "NAPTR":
		r.Data = cfNaptrData(rec)
		r.Name = rec.GetLabelFQDN()
		r.Content = rec.GetRDATA().String()
	case "HTTPS", "SVCB":
		r.Data = cfSvcbData(rec)
		r.Content = rec.GetRDATA().String()
	case "LOC":
		r.Data = cfLocData(rec)
		r.Content = rec.GetRDATA().String()
	default:
		r.Content = rec.GetRDATA().String()
	}

	_, err := c.cfClient.UpdateDNSRecord(context.Background(), cloudflare.ZoneIdentifier(domainID), r)
	return err
}

// change universal ssl state.
func (c *cloudflareProvider) changeUniversalSSL(domainID string, state bool) error {
	_, err := c.cfClient.EditUniversalSSLSetting(context.Background(), domainID, cloudflare.UniversalSSLSetting{Enabled: state})
	return err
}

// get universal ssl state.
func (c *cloudflareProvider) getUniversalSSL(domainID string) (bool, error) {
	result, err := c.cfClient.UniversalSSLSettingDetails(context.Background(), domainID)
	return result.Enabled, err
}

func (c *cloudflareProvider) getSingleRedirects(dc *models.DomainConfig, id string) ([]*models.RecordConfig, error) {
	rules, err := c.cfClient.GetEntrypointRuleset(context.Background(), cloudflare.ZoneIdentifier(id), "http_request_dynamic_redirect")
	if err != nil {
		if _, ok := errors.AsType[*cloudflare.NotFoundError](err); ok {
			return nil, nil
		}
		return nil, fmt.Errorf("failed fetching redirect rule list cloudflare: %w (%T)", err, err)
	}

	var recs models.Records
	for _, pr := range rules.Rules {
		thisPr := pr

		// Extract the valuables from the rule, use it to make the sr:
		srName := pr.Description
		srWhen := pr.Expression
		srThen := pr.ActionParameters.FromValue.TargetURL.Expression
		code := uint16(pr.ActionParameters.FromValue.StatusCode)

		// Make the record:
		rec, err := dc.NewRecordConfig("@", 1, privatetypes.TypeCLOUDFLAREAPISINGLEREDIRECT, srName, code, srWhen, srThen)
		if err != nil {
			return nil, err
		}
		rec.Original = thisPr

		// Store the IDs. These will be needed for update/delete operations.
		sr := rec.AsCLOUDFLAREAPISINGLEREDIRECT()
		sr.RT_SRRRulesetID = rules.ID
		sr.RT_SRRRulesetRuleID = pr.ID
		rec.SetRDATA(sr)

		recs = append(recs, rec)
	}

	return recs, nil
}

func (c *cloudflareProvider) createSingleRedirect(domainID string, cfr privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT) error {
	newSingleRedirectRulesActionParameters := cloudflare.RulesetRuleActionParameters{}
	newSingleRedirectRule := cloudflare.RulesetRule{}
	newSingleRedirectRules := []cloudflare.RulesetRule{}
	newSingleRedirectRules = append(newSingleRedirectRules, newSingleRedirectRule)
	newSingleRedirect := cloudflare.UpdateEntrypointRulesetParams{}

	// Preserve query string if there isn't one in the replacement.
	preserveQueryString := !strings.Contains(cfr.SRThen, "?")

	newSingleRedirectRulesActionParameters.FromValue = &cloudflare.RulesetRuleActionParametersFromValue{}
	// Redirect status code
	newSingleRedirectRulesActionParameters.FromValue.StatusCode = uint16(cfr.Code)
	// Incoming request expression
	newSingleRedirectRules[0].Expression = cfr.SRWhen
	// Redirect expression
	newSingleRedirectRulesActionParameters.FromValue.TargetURL.Expression = cfr.SRThen
	// Redirect name
	newSingleRedirectRules[0].Description = cfr.SRName

	// Rule action, should always be redirect in this case
	newSingleRedirectRules[0].Action = "redirect"
	// Phase should always be http_request_dynamic_redirect
	newSingleRedirect.Phase = "http_request_dynamic_redirect"

	// Assigns the values in the nested structs
	newSingleRedirectRulesActionParameters.FromValue.PreserveQueryString = &preserveQueryString
	newSingleRedirectRules[0].ActionParameters = &newSingleRedirectRulesActionParameters

	// Get a list of current redirects so that the new redirect get appended to it
	rules, err := c.cfClient.GetEntrypointRuleset(context.Background(), cloudflare.ZoneIdentifier(domainID), "http_request_dynamic_redirect")
	var e *cloudflare.NotFoundError
	if err != nil && !errors.As(err, &e) {
		return fmt.Errorf("failed fetching redirect rule list cloudflare: %w", err)
	}
	newSingleRedirect.Rules = newSingleRedirectRules
	newSingleRedirect.Rules = append(rules.Rules, newSingleRedirect.Rules...)

	_, err = c.cfClient.UpdateEntrypointRuleset(context.Background(), cloudflare.ZoneIdentifier(domainID), newSingleRedirect)

	return err
}

func (c *cloudflareProvider) deleteSingleRedirects(domainID string, cfr privatetypesrdata.CLOUDFLAREAPISINGLEREDIRECT) error {
	err := c.cfClient.DeleteRulesetRule(context.Background(), cloudflare.ZoneIdentifier(domainID), cloudflare.DeleteRulesetRuleParams{
		RulesetID:     cfr.RT_SRRRulesetID,
		RulesetRuleID: cfr.RT_SRRRulesetRuleID,
	},
	)
	// NB(tlim): Yuck. This returns an error even when it is successful. Dig into the JSON for the real status.
	if strings.Contains(err.Error(), `"success": true,`) {
		return nil
	}

	return err
}

func (c *cloudflareProvider) updateSingleRedirect(domainID string, oldrec, newrec *models.RecordConfig) error {
	if err := c.deleteSingleRedirects(domainID, oldrec.AsCLOUDFLAREAPISINGLEREDIRECT()); err != nil {
		return err
	}
	return c.createSingleRedirect(domainID, newrec.AsCLOUDFLAREAPISINGLEREDIRECT())
}

func (c *cloudflareProvider) getWorkerRoutes(id string, dc *models.DomainConfig) ([]*models.RecordConfig, error) {
	res, err := c.cfClient.ListWorkerRoutes(context.Background(), cloudflare.ZoneIdentifier(id), cloudflare.ListWorkerRoutesParams{})
	if err != nil {
		return nil, fmt.Errorf("failed fetching worker route list cloudflare: %w", err)
	}

	var recs models.Records
	for _, pr := range res.Routes {
		thisPr := pr

		r, err := dc.NewRecordConfig("@", 1, privatetypes.TypeCFWORKERROUTE, pr.Pattern, pr.ScriptName)
		if err != nil {
			return nil, err
		}
		r.Original = thisPr

		recs = append(recs, r)
	}
	return recs, nil
}

func (c *cloudflareProvider) deleteWorkerRoute(recordID, domainID string) error {
	_, err := c.cfClient.DeleteWorkerRoute(context.Background(), cloudflare.ZoneIdentifier(domainID), recordID)
	return err
}

func (c *cloudflareProvider) updateWorkerRoute(recordID, domainID string, rd dnsv2.RDATA) error {
	if err := c.deleteWorkerRoute(recordID, domainID); err != nil {
		return err
	}
	return c.createWorkerRoute(domainID, rd)
}

func (c *cloudflareProvider) createWorkerRoute(domainID string, rd dnsv2.RDATA) error {
	rdwr := rd.(privatetypesrdata.CFWORKERROUTE)
	wr := cloudflare.CreateWorkerRouteParams{
		Pattern: rdwr.When,
		Script:  rdwr.Then,
	}

	_, err := c.cfClient.CreateWorkerRoute(context.Background(), cloudflare.ZoneIdentifier(domainID), wr)
	return err
}

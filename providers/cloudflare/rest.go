package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v4/pkg/rtypecontrol"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
	"github.com/DNSControl/dnscontrol/v4/providers/cloudflare/rtypes/cfsingleredirect"
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
func (c *cloudflareProvider) getRecordsForDomain(id string, domain string) ([]*models.RecordConfig, error) {
	records := []*models.RecordConfig{}
	rrs, _, err := c.cfClient.ListDNSRecords(context.Background(), cloudflare.ZoneIdentifier(id), cloudflare.ListDNSRecordsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed fetching record list from cloudflare(%q): %w", c.cfClient.APIEmail, err)
	}
	for _, rec := range rrs {
		rt, err := c.nativeToRecord(domain, rec)
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
	return &cfRecData{
		Algorithm: rec.DnskeyAlgorithm,
		Flags:     rec.DnskeyFlags,
		Protocol:  rec.DnskeyProtocol,
		PublicKey: rec.DnskeyPublicKey,
	}
}

func cfDSData(rec *models.RecordConfig) *cfRecData {
	return &cfRecData{
		KeyTag:     rec.DsKeyTag,
		Algorithm:  rec.DsAlgorithm,
		DigestType: rec.DsDigestType,
		Digest:     rec.DsDigest,
	}
}

func cfSrvData(rec *models.RecordConfig) *cfRecData {
	serverParts := strings.Split(rec.GetLabelFQDN(), ".")
	c := &cfRecData{
		Service:  serverParts[0],
		Proto:    serverParts[1],
		Name:     strings.Join(serverParts[2:], "."),
		Port:     rec.SrvPort,
		Priority: rec.SrvPriority,
		Weight:   rec.SrvWeight,
	}
	c.Target = cfTarget(rec.GetTargetField())
	return c
}

func cfCaaData(rec *models.RecordConfig) *cfRecData {
	return &cfRecData{
		Tag:   rec.CaaTag,
		Flags: uint16(rec.CaaFlag),
		Value: rec.GetTargetField(),
	}
}

func cfTlsaData(rec *models.RecordConfig) *cfRecData {
	return &cfRecData{
		Usage:        rec.TlsaUsage,
		Selector:     rec.TlsaSelector,
		MatchingType: rec.TlsaMatchingType,
		Certificate:  rec.GetTargetField(),
	}
}

func cfSshfpData(rec *models.RecordConfig) *cfRecData {
	return &cfRecData{
		Algorithm:   rec.SshfpAlgorithm,
		HashType:    rec.SshfpFingerprint,
		Fingerprint: rec.GetTargetField(),
	}
}

func cfSvcbData(rec *models.RecordConfig) *cfRecData {
	return &cfRecData{
		Priority: rec.SvcPriority,
		Target:   cfTarget(rec.GetTargetField()),
		Value:    rec.SvcParams,
	}
}

func cfLocData(rec *models.RecordConfig) *cfRecData {
	latDir, latDeg, latMin, latSec := models.ReverseLatitude(rec.LocLatitude)
	longDir, longDeg, longMin, longSec := models.ReverseLongitude(rec.LocLongitude)

	return &cfRecData{
		Altitude:      models.ReverseAltitude(rec.LocAltitude),
		LatDegrees:    latDeg,
		LatDirection:  latDir,
		LatMinutes:    latMin,
		LatSeconds:    latSec,
		LongDegrees:   longDeg,
		LongDirection: longDir,
		LongMinutes:   longMin,
		LongSeconds:   longSec,
		PrecisionHorz: models.ReverseENotationInt(rec.LocHorizPre),
		PrecisionVert: models.ReverseENotationInt(rec.LocVertPre),
		Size:          models.ReverseENotationInt(rec.LocSize),
	}
}

func cfNaptrData(rec *models.RecordConfig) *cfNaptrRecData {
	return &cfNaptrRecData{
		Flags:       rec.NaptrFlags,
		Order:       rec.NaptrOrder,
		Preference:  rec.NaptrPreference,
		Regex:       rec.NaptrRegexp,
		Replacement: rec.GetTargetField(),
		Service:     rec.NaptrService,
	}
}

func (c *cloudflareProvider) createRecDiff2(rec *models.RecordConfig, domainID string, msg string) []*models.Correction {
	content := rec.GetTargetField()
	if rec.Metadata[metaOriginalIP] != "" {
		content = rec.Metadata[metaOriginalIP]
	}
	prio := ""
	switch rec.Type {
	case "MX":
		prio = fmt.Sprintf(" %d ", rec.MxPreference)
	case "TXT":
		content = txtutil.EncodeQuoted(rec.GetTargetTXTJoined())
	case "DS":
		content = fmt.Sprintf("%d %d %d %s", rec.DsKeyTag, rec.DsAlgorithm, rec.DsDigestType, rec.DsDigest)
	}
	if msg == "" {
		msg = fmt.Sprintf("CREATE record: %s %s %d%s %s", rec.GetLabel(), rec.Type, rec.TTL, prio, content)
	}
	if rec.Metadata[metaProxy] == "on" || rec.Metadata[metaProxy] == "full" {
		msg = msg + fmt.Sprintf("\nACTIVATE PROXY for new record %s %s %d %s", rec.GetLabel(), rec.Type, rec.TTL, rec.GetTargetField())
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
				Priority: &rec.MxPreference,
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
		ID:       recID,
		Proxied:  &proxied,
		Name:     rec.GetLabel(),
		Type:     rec.Type,
		Content:  rec.GetTargetField(),
		Priority: &rec.MxPreference,
		TTL:      int(rec.TTL),
	}

	// Handle CNAME flattening setting
	if rec.Type == "CNAME" {
		flatten := rec.Metadata[metaCNAMEFlatten] == "on"
		r.Settings = cloudflare.DNSRecordSettings{FlattenCNAME: &flatten}
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
		r.Content = txtutil.EncodeQuoted(rec.GetTargetTXTJoined())
	case "SRV":
		r.Data = cfSrvData(rec)
		r.Name = rec.GetLabelFQDN()
	case "CAA":
		r.Data = cfCaaData(rec)
		r.Name = rec.GetLabelFQDN()
		r.Content = ""
	case "TLSA":
		r.Data = cfTlsaData(rec)
		r.Name = rec.GetLabelFQDN()
	case "SSHFP":
		r.Data = cfSshfpData(rec)
		r.Name = rec.GetLabelFQDN()
	case "DNSKEY":
		r.Data = cfDnskeyData(rec)
		r.Content = ""
	case "DS":
		r.Data = cfDSData(rec)
		r.Content = ""
	case "NAPTR":
		r.Data = cfNaptrData(rec)
		r.Name = rec.GetLabelFQDN()
	case "HTTPS", "SVCB":
		r.Data = cfSvcbData(rec)
	case "LOC":
		r.Data = cfLocData(rec)
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

func (c *cloudflareProvider) getSingleRedirects(id string, domain string) ([]*models.RecordConfig, error) {
	rules, err := c.cfClient.GetEntrypointRuleset(context.Background(), cloudflare.ZoneIdentifier(id), "http_request_dynamic_redirect")
	if err != nil {
		var e *cloudflare.NotFoundError
		if errors.As(err, &e) {
			return []*models.RecordConfig{}, nil
		}
		return nil, fmt.Errorf("failed fetching redirect rule list cloudflare: %w (%T)", err, err)
	}

	recs := []*models.RecordConfig{}
	for _, pr := range rules.Rules {
		thisPr := pr

		// Extract the valuables from the rule, use it to make the sr:
		srName := pr.Description
		srWhen := pr.Expression
		srThen := pr.ActionParameters.FromValue.TargetURL.Expression
		code := uint16(pr.ActionParameters.FromValue.StatusCode)

		rec, err := rtypecontrol.NewRecordConfigFromRaw(rtypecontrol.FromRawOpts{
			Type: "CLOUDFLAREAPI_SINGLE_REDIRECT",
			TTL:  1,
			Args: []any{srName, code, srWhen, srThen},
			DCN:  domaintags.MakeDomainNameVarieties(domain),
		})
		if err != nil {
			return nil, err
		}
		rec.Original = thisPr

		// Store the IDs. These will be needed for update/delete operations.
		sr := rec.F.(*cfsingleredirect.SingleRedirectConfig)
		sr.SRRRulesetID = rules.ID
		sr.SRRRulesetRuleID = pr.ID

		recs = append(recs, rec)
	}

	return recs, nil
}

func (c *cloudflareProvider) createSingleRedirect(domainID string, cfr cfsingleredirect.SingleRedirectConfig) error {
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

func (c *cloudflareProvider) deleteSingleRedirects(domainID string, cfr cfsingleredirect.SingleRedirectConfig) error {
	err := c.cfClient.DeleteRulesetRule(context.Background(), cloudflare.ZoneIdentifier(domainID), cloudflare.DeleteRulesetRuleParams{
		RulesetID:     cfr.SRRRulesetID,
		RulesetRuleID: cfr.SRRRulesetRuleID,
	},
	)
	// NB(tlim): Yuck. This returns an error even when it is successful. Dig into the JSON for the real status.
	if strings.Contains(err.Error(), `"success": true,`) {
		return nil
	}

	return err
}

func (c *cloudflareProvider) updateSingleRedirect(domainID string, oldrec, newrec *models.RecordConfig) error {
	if err := c.deleteSingleRedirects(domainID, *oldrec.F.(*cfsingleredirect.SingleRedirectConfig)); err != nil {
		return err
	}
	return c.createSingleRedirect(domainID, *newrec.F.(*cfsingleredirect.SingleRedirectConfig))
}

func (c *cloudflareProvider) getWorkerRoutes(id string, domain string) ([]*models.RecordConfig, error) {
	res, err := c.cfClient.ListWorkerRoutes(context.Background(), cloudflare.ZoneIdentifier(id), cloudflare.ListWorkerRoutesParams{})
	if err != nil {
		return nil, fmt.Errorf("failed fetching worker route list cloudflare: %w", err)
	}

	recs := []*models.RecordConfig{}
	for _, pr := range res.Routes {
		thisPr := pr
		r := &models.RecordConfig{
			Type:     "WORKER_ROUTE",
			Original: thisPr,
			TTL:      1,
		}
		r.SetLabel("@", domain)
		err := r.SetTarget(fmt.Sprintf("%s,%s", // $PATTERN,$SCRIPT
			pr.Pattern,
			pr.ScriptName))
		if err != nil {
			return nil, err
		}

		recs = append(recs, r)
	}
	return recs, nil
}

func (c *cloudflareProvider) deleteWorkerRoute(recordID, domainID string) error {
	_, err := c.cfClient.DeleteWorkerRoute(context.Background(), cloudflare.ZoneIdentifier(domainID), recordID)
	return err
}

func (c *cloudflareProvider) updateWorkerRoute(recordID, domainID string, target string) error {
	if err := c.deleteWorkerRoute(recordID, domainID); err != nil {
		return err
	}
	return c.createWorkerRoute(domainID, target)
}

func (c *cloudflareProvider) createWorkerRoute(domainID string, target string) error {
	// $PATTERN,$SCRIPT
	parts := strings.Split(target, ",")
	if len(parts) != 2 {
		return fmt.Errorf("unexpected target: '%s' (expected: 'PATTERN,SCRIPT')", target)
	}
	wr := cloudflare.CreateWorkerRouteParams{
		Pattern: parts[0],
		Script:  parts[1],
	}

	_, err := c.cfClient.CreateWorkerRoute(context.Background(), cloudflare.ZoneIdentifier(domainID), wr)
	return err
}

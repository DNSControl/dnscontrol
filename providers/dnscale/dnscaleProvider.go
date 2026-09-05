package dnscale

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	privatetypes "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

/*

DNScale API DNS provider:

Info required in `creds.json`:
   - api_key

*/

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Cannot(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "DNSCALE"
	const providerMaintainer = "@dnscale-ops"
	fns := providers.DspFuncs{
		Initializer:   NewProvider,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "dnscale",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/dnscale",
		PortalURL:   "https://dnscale.net/", // TODO: Verify
		Fields: []providers.CredsField{
			{
				Key:      "api_key",
				Label:    "API key",
				Help:     "Your dnscale API key.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

// dnscaleProvider represents the DNScale DNSServiceProvider.
type dnscaleProvider struct {
	client  *http.Client
	apiKey  string
	baseURL string
}

// Zone represents a DNScale zone.
type Zone struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Region      string   `json:"region"`
	TTL         int      `json:"ttl"`
	Status      string   `json:"status"`
	Nameservers []string `json:"nameservers"`
}

// Record represents a DNScale DNS record.
type Record struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
	Disabled bool   `json:"disabled"`
}

// apiResponse represents a generic API response.
type apiResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  string          `json:"error,omitempty"`
}

// zonesResponse represents the zones list response.
type zonesResponse struct {
	Zones []Zone `json:"zones"`
}

// recordsResponse represents the records list response.
type recordsResponse struct {
	Records []Record `json:"records"`
}

// NewProvider initializes a DNScale DNSServiceProvider.
func NewProvider(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	apiKey := m["api_key"]
	if apiKey == "" {
		return nil, errors.New("missing DNScale api_key")
	}

	baseURL := m["api_url"]
	if baseURL == "" {
		baseURL = "https://api.dnscale.eu/v1"
	}

	provider := &dnscaleProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:  apiKey,
		baseURL: baseURL,
	}

	// Validate credentials by listing zones
	_, err := provider.listZones()
	if err != nil {
		return nil, fmt.Errorf("failed to validate DNScale credentials: %w", err)
	}

	return provider, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (p *dnscaleProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	zone, err := p.getZoneByName(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone %s: %w", domain, err)
	}

	records, err := p.listRecords(zone.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list records for zone %s: %w", domain, err)
	}

	curRecords := make(models.Records, 0, len(records))
	for _, rec := range records {
		// Skip DNScale's own NS records at apex - these are system-managed.
		// Third-party NS records at apex are kept for multi-provider DNS setups.
		if rec.Type == "NS" && (rec.Name == domain+"." || rec.Name == "@") {
			content := strings.TrimSuffix(rec.Content, ".")
			if strings.HasSuffix(content, ".dnscale.eu") || strings.HasSuffix(content, ".dnscale.com") {
				continue
			}
		}
		// Skip SOA records
		if rec.Type == "SOA" {
			continue
		}

		rc, err := toRecordConfig(dc, rec)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record: %w", err)
		}
		curRecords = append(curRecords, rc)
	}

	return curRecords, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (p *dnscaleProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, curRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction

	zone, err := p.getZoneByName(dc.Name)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get zone %s: %w", dc.Name, err)
	}
	zoneID := zone.ID

	changes, actualChangeCount, err := diff2.ByRecord(curRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE:
			rec := fromRecordConfig(change.New[0])
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					return p.createRecord(zoneID, rec)
				},
			})
		case diff2.CHANGE:
			oldRec := change.Old[0].Original.(Record)
			rec := fromRecordConfig(change.New[0])
			corrections = append(corrections, &models.Correction{
				Msg: fmt.Sprintf("%s; DNScale RecordID: %v", change.Msgs[0], oldRec.ID),
				F: func() error {
					return p.updateRecord(zoneID, oldRec.ID, rec)
				},
			})
		case diff2.DELETE:
			oldRec := change.Old[0].Original.(Record)
			corrections = append(corrections, &models.Correction{
				Msg: fmt.Sprintf("%s; DNScale RecordID: %v", change.Msgs[0], oldRec.ID),
				F: func() error {
					return p.deleteRecord(zoneID, oldRec.ID)
				},
			})
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

// GetNameservers returns an empty array because DNScale assigns nameservers
// server-side when a zone is created. Returning empty means dnscontrol won't
// auto-generate NS records for DNScale at the apex.
//
// For multi-provider DNS setups, you must explicitly declare DNScale's
// nameservers so they are included in registrar delegation:
//
//	D("example.com", REG_NAMECHEAP,
//	  DnsProvider(DSP_DNSCALE),
//	  DnsProvider(DSP_CLOUDFLARE),
//	  NAMESERVER("ns1.dnscale.eu"),
//	  NAMESERVER("ns2.dnscale.eu"),
//	  // ...
//	)
func (p *dnscaleProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return []*models.Nameserver{}, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (p *dnscaleProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	_, err := p.getZoneByName(domain)
	if err == nil {
		return nil // Zone exists
	}

	// Zone doesn't exist, create it
	return p.createZone(domain)
}

// API methods

func (p *dnscaleProvider) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "dnscontrol-dnscale")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiResp apiResponse
		if err := json.Unmarshal(respBody, &apiResp); err == nil && apiResp.Error != "" {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, apiResp.Error)
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (p *dnscaleProvider) listZones() ([]Zone, error) {
	respBody, err := p.doRequest(context.Background(), "GET", "/zones?limit=1000", nil)
	if err != nil {
		return nil, err
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var zonesResp zonesResponse
	if err := json.Unmarshal(apiResp.Data, &zonesResp); err != nil {
		return nil, fmt.Errorf("failed to parse zones: %w", err)
	}

	return zonesResp.Zones, nil
}

func (p *dnscaleProvider) getZoneByName(name string) (*Zone, error) {
	zones, err := p.listZones()
	if err != nil {
		return nil, err
	}

	// Normalize domain name (remove trailing dot)
	name = strings.TrimSuffix(name, ".")

	for _, zone := range zones {
		zoneName := strings.TrimSuffix(zone.Name, ".")
		if zoneName == name {
			return &zone, nil
		}
	}

	return nil, fmt.Errorf("zone %s not found", name)
}

func (p *dnscaleProvider) createZone(name string) error {
	reqBody := map[string]any{
		"name":   name,
		"type":   "primary",
		"region": "EU",
	}

	_, err := p.doRequest(context.Background(), "POST", "/zones", reqBody)
	return err
}

func (p *dnscaleProvider) listRecords(zoneID string) ([]Record, error) {
	respBody, err := p.doRequest(context.Background(), "GET", fmt.Sprintf("/zones/%s/records?limit=1000", zoneID), nil)
	if err != nil {
		return nil, err
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var recordsResp recordsResponse
	if err := json.Unmarshal(apiResp.Data, &recordsResp); err != nil {
		return nil, fmt.Errorf("failed to parse records: %w", err)
	}

	return recordsResp.Records, nil
}

func (p *dnscaleProvider) createRecord(zoneID string, rec Record) error {
	reqBody := map[string]any{
		"name":    rec.Name,
		"type":    rec.Type,
		"content": rec.Content,
		"ttl":     rec.TTL,
	}
	if rec.Priority > 0 {
		reqBody["priority"] = rec.Priority
	}

	_, err := p.doRequest(context.Background(), "POST", fmt.Sprintf("/zones/%s/records", zoneID), reqBody)
	return err
}

func (p *dnscaleProvider) updateRecord(zoneID, recordID string, rec Record) error {
	reqBody := map[string]any{
		"name":    rec.Name,
		"type":    rec.Type,
		"content": rec.Content,
		"ttl":     rec.TTL,
	}
	if rec.Priority > 0 {
		reqBody["priority"] = rec.Priority
	}

	_, err := p.doRequest(context.Background(), "PUT", fmt.Sprintf("/zones/%s/records/%s", zoneID, recordID), reqBody)
	return err
}

func (p *dnscaleProvider) deleteRecord(zoneID, recordID string) error {
	_, err := p.doRequest(context.Background(), "DELETE", fmt.Sprintf("/zones/%s/records/%s", zoneID, recordID), nil)
	return err
}

// Record conversion functions

// toRecordConfig converts a DNScale Record to a RecordConfig.
func toRecordConfig(dc *models.DomainConfig, r Record) (*models.RecordConfig, error) {
	// Extract label from full record name
	domain := dc.Name

	name := strings.TrimSuffix(r.Name, ".")
	domainWithDot := domain + "."
	if before, ok := strings.CutSuffix(r.Name, domainWithDot); ok {
		name = before
		name = strings.TrimSuffix(name, ".")
	} else if before, ok := strings.CutSuffix(r.Name, domain); ok {
		name = strings.TrimSuffix(before, ".")
	}
	if name == "" {
		name = "@"
	}
	label := dc.LabelFromShort(name)

	// TODO(tom): The above code can probably be replaced with the following line, but I haven't tested it yet.
	//label := dc.LabelFromFQDNWithDot(r.Name)
	// And if that doesn't work... try this:
	//label := dc.LabelFromFQDNNoDot(r.Name)

	content := r.Content
	ttl := uint32(r.TTL)
	var rc *models.RecordConfig
	var err error

	switch r.Type {
	case "A", "AAAA":
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, content)
	case "CNAME", "NS", "PTR", "ALIAS":
		// TODO(tlim): Test without this "if". It may no longer be needed.

		// Ensure FQDN
		if !strings.HasSuffix(content, ".") {
			content = content + "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, content)
	case "MX":
		// TODO(tlim): Test replacing this with simply:
		// rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, content)

		// DNScale API returns MX as "priority target" in content field
		// If priority field is 0, parse it from content
		priority := uint16(r.Priority)
		target := content
		if priority == 0 && strings.Contains(content, " ") {
			parts := strings.SplitN(content, " ", 2)
			if len(parts) == 2 {
				if p, err := strconv.ParseUint(parts[0], 10, 16); err == nil {
					priority = uint16(p)
					target = parts[1]
				}
			}
		}
		if !strings.HasSuffix(target, ".") {
			target = target + "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, priority, target)
	case "TXT":
		// DNScale returns TXT content without surrounding quotes
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, content)
	case "SRV":
		// DNScale API returns SRV as "priority weight port target"

		// TODO(tlim): If the NEW version works, please remove the commented out "OLD" version.

		// OLD:

		// // Set|TargetSRV expects priority, weight, port, target separately
		// parts := strings.Fields(content)
		// if len(parts) != 4 {
		// 	return nil, fmt.Errorf("SRV value does not contain 4 fields: (%q)", content)
		// }
		// priority, parseErr := strconv.ParseUint(parts[0], 10, 16)
		// if parseErr != nil {
		// 	return nil, fmt.Errorf("SRV priority parse error: %w", parseErr)
		// }
		// weight, parseErr := strconv.ParseUint(parts[1], 10, 16)
		// if parseErr != nil {
		// 	return nil, fmt.Errorf("SRV weight parse error: %w", parseErr)
		// }
		// port, parseErr := strconv.ParseUint(parts[2], 10, 16)
		// if parseErr != nil {
		// 	return nil, fmt.Errorf("SRV port parse error: %w", parseErr)
		// }
		// target := parts[3]
		// if !strings.HasSuffix(target, ".") {
		// 	target = target + "."
		// }
		// rc, err = dc.NewRecordConfig(label, ttl, r.Type, uint16(priority), uint16(weight), uint16(port), target)

		// NEW:
		if !strings.HasSuffix(content, ".") {
			content = content + "."
		}
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, content)

	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, content)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r
	return rc, nil
}

// fromRecordConfig converts a RecordConfig to a DNScale Record.
func fromRecordConfig(rc *models.RecordConfig) Record {
	name := rc.GetLabel()

	// TODO(tlim): Is this needed? Seems like a no-op.
	// DNScale uses "@" for apex
	if name == "@" {
		name = "@"
	}

	var content string
	priority := 0

	switch rc.TypeNum {
	case dnsv2.TypeCNAME, dnsv2.TypeNS, dnsv2.
		// Remove trailing dot for DNScale API
		TypePTR, privatetypes.TypeALIAS:
		content = strings.TrimSuffix(rc.GetRDATA().String(), ".")
	case dnsv2.TypeMX:
		f := rc.AsMX()
		priority = int(f.Preference)
		content = strings.TrimSuffix(f.Mx, ".")
	// case "SRV":
	// 	// TODO(tlim): Remove commented out code if new code works.
	// 	// DNScale API expects full content: "priority weight port target"
	// 	// target := rc.Get|TargetField()
	// 	// if !strings.HasSuffix(target, ".") {
	// 	// 	target = target + "."
	// 	// }
	// 	// content = fmt.Sprintf("%d %d %d %s", rc.SrvPriority, rc.SrvWeight, rc.SrvPort, target)
	// 	content = rc.GetRDATA().String()

	// case "CAA":
	// 	// TODO(tlim): Remove commented out code if new code works.
	// 	//content = fmt.Sprintf("%d %s \"%s\"", rc.CaaFlag, rc.CaaTag, rc.Get|TargetField())
	// 	content = rc.GetRDATA().String()
	// case "TLSA":
	// 	// TODO(tlim): Remove commented out code if new code works.
	// 	//content = fmt.Sprintf("%d %d %d %s", rc.TlsaUsage, rc.TlsaSelector, rc.TlsaMatchingType, rc.Get|TargetField())
	// 	content = rc.GetRDATA().String()
	// case "SSHFP":
	// 	// TODO(tlim): Remove commented out code if new code works.
	// 	//content = fmt.Sprintf("%d %d %s", rc.SshfpAlgorithm, rc.SshfpFingerprint, rc.Get|TargetField())
	// 	content = rc.GetRDATA().String()
	case dnsv2.TypeHTTPS, dnsv2.TypeSVCB:
		// Use GetRDATA().String() which formats SVCB/HTTPS records correctly via miekg/dns
		// DNScale API requires selective quote handling for SVCB params:
		// - alpn="h2,h3" must become alpn=h2,h3 (quotes stripped)
		// - ech="base64..." must keep quotes (required for base64 values)

		content = stripSvcbQuotesExceptEch(rc.GetRDATA().String())
	case dnsv2.TypeTXT:
		content = rc.GetTargetTXTJoined()
	default:
		content = rc.GetRDATA().String()
	}

	return Record{
		Name:     name,
		Type:     rc.Type,
		Content:  content,
		TTL:      int(rc.TTL),
		Priority: priority,
	}
}

// stripSvcbQuotesExceptEch removes quotes from SVCB/HTTPS param values except for 'ech'.
// DNScale API requires: alpn=h2,h3 (no quotes) but ech="base64..." (with quotes).
func stripSvcbQuotesExceptEch(content string) string {
	// Simple approach: temporarily protect ech values, strip all quotes, restore ech quotes
	// This handles patterns like: 1 . alpn="h2,h3" ech="base64..."

	// If no ech param, just strip all quotes
	if !strings.Contains(content, "ech=") {
		return strings.ReplaceAll(content, "\"", "")
	}

	// Find and protect ech value, strip other quotes, then restore
	// Pattern: ech="..." where ... is the base64 value
	result := content
	echStart := strings.Index(result, "ech=\"")
	if echStart == -1 {
		// ech without quotes - nothing special to do
		return strings.ReplaceAll(content, "\"", "")
	}

	// Find the closing quote for ech value
	valueStart := echStart + 5 // len("ech=\"")
	echEnd := strings.Index(result[valueStart:], "\"")
	if echEnd == -1 {
		return strings.ReplaceAll(content, "\"", "")
	}
	echEnd += valueStart

	// Extract parts: before ech, ech value (without quotes), after ech
	beforeEch := result[:echStart]
	echValue := result[valueStart:echEnd]
	afterEch := result[echEnd+1:]

	// Strip quotes from before and after parts
	beforeEch = strings.ReplaceAll(beforeEch, "\"", "")
	afterEch = strings.ReplaceAll(afterEch, "\"", "")

	// Reconstruct with ech value quoted
	return beforeEch + "ech=\"" + echValue + "\"" + afterEch
}

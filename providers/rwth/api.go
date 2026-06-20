package rwth

// The documentation is hosted at https://noc-portal.rz.rwth-aachen.de/dns-admin/en/api_tokens and
// https://blog.rwth-aachen.de/itc/2022/07/13/api-im-dns-admin/

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnsrr"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

const (
	baseURL = "https://noc-portal.rz.rwth-aachen.de/dns-admin/api/v1"
)

// RecordReply represents a DNS Record in an API.
type RecordReply struct {
	ID        int       `json:"id"`
	ZoneID    int       `json:"zone_id"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
	Editable  bool      `json:"editable"`
}

type dnssecKey struct {
	CreatedAt time.Time `json:"created_at"`
}

// dnssecStatus supports both API variants:
// - dnssec: false
// - dnssec: { ...keys... }.
type dnssecStatus struct {
	Enabled        bool       `json:"-"`
	ZoneSigningKey *dnssecKey `json:"zone_signing_key,omitempty"`
	KeySigningKey  *dnssecKey `json:"key_signing_key,omitempty"`
}

func (d *dnssecStatus) UnmarshalJSON(data []byte) error {
	// RWTH may return either a boolean or an object for the same field.
	var enabled bool
	if err := json.Unmarshal(data, &enabled); err == nil {
		*d = dnssecStatus{Enabled: enabled}
		return nil
	}

	var payload struct {
		ZoneSigningKey *dnssecKey `json:"zone_signing_key"`
		KeySigningKey  *dnssecKey `json:"key_signing_key"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("invalid dnssec value: %w", err)
	}

	*d = dnssecStatus{
		Enabled:        true,
		ZoneSigningKey: payload.ZoneSigningKey,
		KeySigningKey:  payload.KeySigningKey,
	}
	return nil
}

type zone struct {
	ID         int          `json:"id"`
	ZoneName   string       `json:"zone_name"`
	Status     string       `json:"status"`
	UpdatedAt  time.Time    `json:"updated_at"`
	LastDeploy time.Time    `json:"last_deploy"`
	Dnssec     dnssecStatus `json:"dnssec"`
}

func checkIsLockedSystemAPIRecord(record RecordReply) error {
	if record.Type == "soa_record" {
		// The upload of a BIND zone file can change the SOA record.
		// Implementing this edge case this is too complex for now.
		return errors.New("SOA records are locked in RWTH zones. They are hence not available for updating")
	}
	return nil
}

func checkIsLockedSystemRecord(record *models.RecordConfig) error {
	if record.Type == "SOA" {
		// The upload of a BIND zone file can change the SOA record.
		// Implementing this edge case this is too complex for now.
		return errors.New("SOA records are locked in RWTH zones. They are hence not available for updating")
	}
	return nil
}

func (api *rwthProvider) createRecord(record *models.RecordConfig) error {
	if err := checkIsLockedSystemRecord(record); err != nil {
		return err
	}

	req := url.Values{}
	req.Set("record_content", api.printRecConfig(*record))
	return api.request("/create_record", "POST", req, nil)
}

func (api *rwthProvider) destroyRecord(record RecordReply) error {
	if err := checkIsLockedSystemAPIRecord(record); err != nil {
		return err
	}
	req := url.Values{}
	req.Set("record_id", strconv.Itoa(record.ID))
	return api.request("/destroy_record", "DELETE", req, nil)
}

func (api *rwthProvider) updateRecord(id int, record models.RecordConfig) error {
	if err := checkIsLockedSystemRecord(&record); err != nil {
		return err
	}
	req := url.Values{}
	req.Set("record_id", strconv.Itoa(id))
	req.Set("record_content", api.printRecConfig(record))
	return api.request("/update_record", "POST", req, nil)
}

func (api *rwthProvider) getAllRecords(domain string) ([]models.RecordConfig, error) {
	zone, err := api.getZone(domain)
	if err != nil {
		return nil, err
	}
	records := make([]models.RecordConfig, 0)
	response := []RecordReply{}
	request := url.Values{}
	request.Set("zone_id", strconv.Itoa(zone.ID))
	if err := api.request("/list_records", "GET", request, &response); err != nil {
		return nil, fmt.Errorf("failed fetching zone records for %q: %w", domain, err)
	}
	for _, apiRecord := range response {
		if checkIsLockedSystemAPIRecord(apiRecord) != nil {
			continue
		}
		dnsRec, err := NewRR(apiRecord.Content) // Parse content as DNS record
		if err != nil {
			return nil, err
		}

		recConfig, err := dnsrr.RRtoRC(dnsRec, domain) // and make it a RC
		if err != nil {
			return nil, err
		}
		recConfig.Original = apiRecord // but keep our ApiRecord as the original

		records = append(records, recConfig)
	}
	return records, nil
}

func (api *rwthProvider) getAllZones() error {
	if api.zones != nil {
		return nil
	}
	zones := map[string]zone{}
	response := &[]zone{}
	if err := api.request("/list_zones", "GET", url.Values{}, response); err != nil {
		return fmt.Errorf("failed fetching zones: %w", err)
	}
	for _, zone := range *response {
		zones[zone.ZoneName] = zone
	}
	api.zones = zones
	return nil
}

func (api *rwthProvider) getZone(name string) (*zone, error) {
	if err := api.getAllZones(); err != nil {
		return nil, err
	}
	zone, ok := api.zones[name]
	if !ok {
		return nil, fmt.Errorf("%q is not a zone in this RWTH account", name)
	}
	return &zone, nil
}

// Deploy the zone.
func (api *rwthProvider) deployZone(domain string) error {
	zone, err := api.getZone(domain)
	if err != nil {
		return err
	}
	req := url.Values{}
	req.Set("zone_id", strconv.Itoa(zone.ID))
	return api.request("/deploy_zone", "POST", req, nil)
}

// Send a request.
func (api *rwthProvider) request(endpoint string, method string, request url.Values, target any) error {
	requestBody := strings.NewReader(request.Encode())
	req, err := http.NewRequest(method, baseURL+endpoint, requestBody)
	if err != nil {
		return err
	}
	req.Header.Add("PRIVATE-TOKEN", api.apiToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	cleanupResponseBody := func() {
		err := resp.Body.Close()
		if err != nil {
			printer.Printf("failed closing response body: %q\n", err)
		}
	}

	defer cleanupResponseBody()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		printer.Printf("%s", string(data))
		return fmt.Errorf("bad status code from RWTH: %d not 200", resp.StatusCode)
	}
	if target == nil {
		return nil
	}
	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(target)
}

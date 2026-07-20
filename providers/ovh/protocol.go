package ovh

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
	"github.com/ovh/go-ovh/ovh"
)

// Void an empty structure.
type Void struct{}

// fetchDomainList gets list of zones for account.
func (c *ovhProvider) fetchZones() error {
	if c.zones != nil {
		return nil
	}
	c.zones = map[string]bool{}

	var response []string

	err := c.client.CallAPI("GET", "/domain/zone", nil, &response, true)
	if err != nil {
		return err
	}

	for _, d := range response {
		c.zones[d] = true
	}
	return nil
}

// Zone describes the attributes of a DNS zone.
type Zone struct {
	DNSSecSupported bool     `json:"dnssecSupported"`
	HasDNSAnycast   bool     `json:"hasDNSAnycast,omitempty"`
	NameServers     []string `json:"nameServers"`
	LastUpdate      string   `json:"lastUpdate,omitempty"`
}

// get info about a zone.
func (c *ovhProvider) fetchZone(fqdn string) (*Zone, error) {
	var response Zone

	err := c.client.CallAPI("GET", "/domain/zone/"+fqdn, nil, &response, true)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// Record describes a DNS record.
type Record struct {
	Target    string `json:"target,omitempty"`
	Zone      string `json:"zone,omitempty"`
	TTL       uint32 `json:"ttl,omitempty"`
	FieldType string `json:"fieldType,omitempty"`
	ID        int64  `json:"id,omitempty"`
	SubDomain string `json:"subDomain,omitempty"`
}

func (c *ovhProvider) fetchRecords(fqdn string) ([]*Record, error) {
	var recordIds []int

	err := c.client.CallAPI("GET", "/domain/zone/"+fqdn+"/record", nil, &recordIds, true)
	if err != nil {
		return nil, err
	}

	records := make([]*Record, len(recordIds))
	for i, id := range recordIds {
		r, err := c.fetchRecord(fqdn, id)
		if err != nil {
			return nil, err
		}
		records[i] = r
	}

	return records, nil
}

func (c *ovhProvider) fetchRecord(fqdn string, id int) (*Record, error) {
	var response Record

	err := c.client.CallAPI("GET", fmt.Sprintf("/domain/zone/%s/record/%d", fqdn, id), nil, &response, true)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// Returns a function that can be invoked to delete a record in a zone.
func (c *ovhProvider) deleteRecordFunc(id int64, fqdn string) func() error {
	return func() error {
		err := c.client.CallAPI("DELETE", fmt.Sprintf("/domain/zone/%s/record/%d", fqdn, id), nil, nil, true)
		if err != nil {
			return err
		}
		return nil
	}
}

// Returns a function that can be invoked to create a record in a zone.
func (c *ovhProvider) createRecordFunc(rc *models.RecordConfig, fqdn string) func() error {
	return func() error {
		recordType := rc.Type
		if nativeType, ok := rc.Metadata["create_ovh_native_record"]; ok {
			recordType = nativeType
		}
		record := Record{
			SubDomain: dnsutilv1.TrimDomainName(rc.GetLabelFQDN(), fqdn),
			FieldType: recordType,
			Target:    rc.GetTargetCombined(),
			TTL:       rc.TTL,
		}
		if record.SubDomain == "@" {
			record.SubDomain = ""
		}

		// note that we never create OVH custom TXT records such as DKIM, SPF or DMARC, instead we prefer to
		// use regular TXT records, unless the record is anotated with the `create_ovh_native_record` metadata

		err := adaptNativeRecord(&record)
		if err != nil {
			return err
		}

		var response Record
		err = c.client.CallAPI("POST", fmt.Sprintf("/domain/zone/%s/record", fqdn), &record, &response, true)
		return err
	}
}

// Returns a function that can be invoked to update a record in a zone.
func (c *ovhProvider) updateRecordFunc(old *Record, rc *models.RecordConfig, fqdn string) func() error {
	return func() error {
		recordType := rc.Type

		if rc.Type != "TXT" && (old.FieldType == "DKIM" || old.FieldType == "SPF" || old.FieldType == "DMARC") {
			return fmt.Errorf("OVH doesn't allow to change %s native type to a non TXT type - delete the record manually and run dnscontrol again", old.FieldType)
		}

		if old.FieldType == "DKIM" || old.FieldType == "SPF" || old.FieldType == "DMARC" {
			recordType = old.FieldType
		}

		record := Record{
			SubDomain: rc.GetLabel(),
			FieldType: recordType,
			Target:    rc.GetTargetCombined(),
			TTL:       rc.TTL,
			Zone:      fqdn,
			ID:        old.ID,
		}
		if record.SubDomain == "@" {
			record.SubDomain = ""
		}

		err := adaptNativeRecord(&record)
		if err != nil {
			return err
		}

		// Native DKIM, SPF or DMARC record field type created through the OVH UI shouldn't be updated
		// or  we get the "Try to alter read-only properties: fieldType" error
		switch old.FieldType {
		case "DKIM", "SPF", "DMARC":
			record.FieldType = ""
		}

		err = c.client.CallAPI("PUT", fmt.Sprintf("/domain/zone/%s/record/%d", fqdn, old.ID), &record, &Void{}, true)

		return err
	}
}

// adaptNativeRecord adapts the record for native OVH types such as DMARC or DKIM.
func adaptNativeRecord(r *Record) error {
	// OVH needs DMARC and DKIM to be "unquoted"
	if r.FieldType == "DMARC" || r.FieldType == "DKIM" {
		// make sure target is fully unquoted to prevent "Invalid subfield found in DMARC" error
		r.Target = models.StripQuotes(r.Target)
	}
	// DMARC record can be created only for subdomains starting with`_dmarc`
	if r.FieldType == "DMARC" && !strings.HasPrefix(r.SubDomain, "_dmarc") {
		return fmt.Errorf("native OVH DMARC record requires subdomain to always start with _dmarc, %s given", r.SubDomain)
	}
	return nil
}

// refreshZone initiates a refresh task on OVHs backend.
func (c *ovhProvider) refreshZone(fqdn string) error {
	return c.client.CallAPI("POST", fmt.Sprintf("/domain/zone/%s/refresh", fqdn), nil, &Void{}, true)
}

// fetch the NS OVH attributed to this zone (which is distinct from fetchRealNS which
// get the exact NS stored at the registrar.
func (c *ovhProvider) fetchZoneNS(fqdn string) ([]string, error) {
	zone, err := c.fetchZone(fqdn)
	if err != nil {
		return nil, err
	}

	return zone.NameServers, nil
}

// Fetch first the registrar NS, if none found, return the zone defined NS.
func (c *ovhProvider) fetchNS(fqdn string) ([]string, error) {
	ns, err := c.fetchRegistrarNS(fqdn)
	if err != nil {
		return nil, err
	}

	if len(ns) == 0 {
		ns, err = c.fetchZoneNS(fqdn)
		if err != nil {
			return nil, err
		}
	}
	return ns, nil
}

// CurrentNameServer stores information about nameservers.
type CurrentNameServer struct {
	ToDelete bool   `json:"toDelete,omitempty"`
	IP       string `json:"ip,omitempty"`
	IsUsed   bool   `json:"isUsed,omitempty"`
	ID       int    `json:"id,omitempty"`
	Host     string `json:"host,omitempty"`
}

// Retrieve the NS currently being deployed to the registrar.
func (c *ovhProvider) fetchRegistrarNS(fqdn string) ([]string, error) {
	var nameServersID []int
	err := c.client.CallAPI("GET", "/domain/"+fqdn+"/nameServer", nil, &nameServersID, true)
	if err != nil {
		var apiError *ovh.APIError
		if errors.As(err, &apiError) && apiError.Code == 404 {
			return []string{}, nil
		}
		return nil, err
	}

	var nameServers []string
	for _, id := range nameServersID {
		var ns CurrentNameServer
		err = c.client.CallAPI("GET", fmt.Sprintf("/domain/%s/nameServer/%d", fqdn, id), nil, &ns, true)
		if err != nil {
			return nil, err
		}

		// skip NS that we asked for deletion
		if ns.ToDelete {
			continue
		}
		nameServers = append(nameServers, ns.Host)
	}

	return nameServers, nil
}

// DomainNS describes a domain's NS in ovh's protocol.
type DomainNS struct {
	Host string `json:"host,omitempty"`
	IP   string `json:"ip,omitempty"`
}

// UpdateNS describes a list of nameservers in ovh's protocol.
type UpdateNS struct {
	NameServers []DomainNS `json:"nameServers"`
}

// Task describes a task in ovh's protocol.
type Task struct {
	Function      string `json:"function,omitempty"`
	Status        string `json:"status,omitempty"`
	CanAccelerate bool   `json:"canAccelerate,omitempty"`
	LastUpdate    string `json:"lastUpdate,omitempty"`
	CreationDate  string `json:"creationDate,omitempty"`
	Comment       string `json:"comment,omitempty"`
	TodoDate      string `json:"todoDate,omitempty"`
	ID            int64  `json:"id,omitempty"`
	CanCancel     bool   `json:"canCancel,omitempty"`
	DoneDate      string `json:"doneDate,omitempty"`
	CanRelaunch   bool   `json:"canRelaunch,omitempty"`
}

// Domain describes a domain in ovh's protocol.
type Domain struct {
	NameServerType     string `json:"nameServerType,omitempty"`
	TransferLockStatus string `json:"transferLockStatus,omitempty"`
}

func (c *ovhProvider) updateNS(fqdn string, ns []string) error {
	// we first need to make sure we can edit the NS
	// by default zones are in "hosted" mode meaning they default
	// to OVH default NS. In this mode, the NS can't be updated.
	domain := Domain{NameServerType: "external"}
	err := c.client.CallAPI("PUT", "/domain/"+fqdn, &domain, &Void{}, true)
	if err != nil {
		return err
	}

	var newNs []DomainNS
	for _, n := range ns {
		newNs = append(newNs, DomainNS{
			Host: n,
		})
	}

	update := UpdateNS{
		NameServers: newNs,
	}
	var task Task
	err = c.client.CallAPI("POST", fmt.Sprintf("/domain/%s/nameServers/update", fqdn), &update, &task, true)
	if err != nil {
		return err
	}

	if task.Status == "error" {
		return fmt.Errorf("API error while updating ns for %s: %s", fqdn, task.Comment)
	}

	// we don't wait for the task execution. One of the reason is that
	// NS modification can take time in the registrar, the other is that every task
	// in OVH is usually executed a few minutes after they have been registered.
	// We count on the fact that `GetNameservers` uses the registrar API to get
	// a coherent view (including pending modifications) of the registered NS.

	return nil
}

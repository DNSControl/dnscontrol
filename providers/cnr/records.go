package cnr

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

// Record covers an individual DNS resource record.
type Record struct {
	// DomainName is the zone that the record belongs to.
	DomainName string
	// Host is the hostname relative to the zone: e.g. for a record for blog.example.org, domain would be "example.org" and host would be "blog".
	// An apex record would be specified by either an empty host "" or "@".
	// A SRV record would be specified by "_{service}._{protocol}.{host}": e.g. "_sip._tcp.phone" for _sip._tcp.phone.example.org.
	Host string
	// FQDN is the Fully Qualified Domain Name. It is the combination of the host and the domain name. It always ends in a ".". FQDN is ignored in CreateRecord, specify via the Host field instead.
	Fqdn string
	// Type is the DNS record type (e.g. A, AAAA, CNAME, MX, LOC, SVCB, etc.).
	Type string
	// Answer is either the IP address for A or AAAA records; the target for ANAME, CNAME, MX, or NS records; the text for TXT records.
	// For SRV records, answer has the following format: "{weight} {port} {target}" e.g. "1 5061 sip.example.org".
	Answer string
	// TTL is the time this record can be cached for in seconds.
	TTL uint32
	// Priority is only required for MX and SRV records, it is ignored for all others.
	Priority uint32
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (n *Client) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	return n.getRecords(dc)
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (n *Client) GetZoneRecordsCorrections(dc *models.DomainConfig, actual models.Records) ([]*models.Correction, int, error) {
	for _, rc := range actual {
		if rc.Type == "SVCB" {
			rc.SvcParams = strings.Join(strings.Fields(rc.SvcParams), " ")
		}
	}
	for _, rc := range dc.Records {
		if rc.Type != "SVCB" {
			continue
		}
		fields := strings.Fields(rc.SvcParams)
		params := make([]string, 0, len(fields))
		for _, field := range fields {
			key, value, _ := strings.Cut(field, "=")
			if strings.EqualFold(strings.TrimSpace(key), "ech") && strings.Trim(value, `"`) == "IGNORE" {
				continue
			}
			params = append(params, field)
		}
		rc.SvcParams = strings.Join(params, " ")
	}

	var aliasSkip *models.Correction
	hasAlias := false
	for _, rc := range dc.Records {
		if rc.Type == "ALIAS" {
			hasAlias = true
			break
		}
	}
	if !hasAlias {
		for _, rc := range actual {
			if rc.Type == "ALIAS" {
				hasAlias = true
				break
			}
		}
	}
	if hasAlias {
		signed, err := n.isZoneSigned(dc.Name)
		if err != nil {
			return nil, 0, err
		}
		if signed {
			skipped := 0
			desired := make(models.Records, 0, len(dc.Records))
			for _, rc := range dc.Records {
				if rc.Type == "ALIAS" {
					skipped++
					continue
				}
				desired = append(desired, rc)
			}
			dc.Records = desired

			filteredActual := make(models.Records, 0, len(actual))
			for _, rc := range actual {
				if rc.Type != "ALIAS" {
					filteredActual = append(filteredActual, rc)
				}
			}
			actual = filteredActual

			if skipped != 0 {
				aliasSkip = &models.Correction{
					Msg: fmt.Sprintf("SKIP ALIAS records in DNSSEC-signed CNR zone %s", dc.Name),
					F:   func() error { return nil },
				}
			}
		}
	}
	toReport, create, del, mod, actualChangeCount, err := diff.NewCompat(dc).IncrementalDiff(actual)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)
	if aliasSkip != nil {
		corrections = append(corrections, aliasSkip)
	}

	buf := &bytes.Buffer{}
	// Print a list of changes. Generate an actual change that is the zone
	changes := false
	var builder strings.Builder
	params := map[string]any{}
	delrridx := 0
	addrridx := 0

	for _, cre := range create {
		changes = true
		fmt.Fprintln(buf, cre)
		newRecordString, err := n.createRecordString(cre.Desired, dc.Name)
		if err != nil {
			return corrections, 0, err
		}
		key := fmt.Sprintf("ADDRR%d", addrridx)
		params[key] = newRecordString
		fmt.Fprintf(&builder, "\033[32m+ %s = %s\033[0m\n", key, newRecordString)
		addrridx++
	}
	for _, d := range del {
		changes = true
		fmt.Fprintln(buf, d)
		key := fmt.Sprintf("DELRR%d", delrridx)
		oldRecordString := d.Existing.Original.(string)
		params[key] = oldRecordString
		fmt.Fprintf(&builder, "\033[31m- %s = %s\033[0m\n", key, oldRecordString)
		delrridx++
	}
	for _, chng := range mod {
		changes = true
		fmt.Fprintln(buf, chng)
		// old record deletion
		key := fmt.Sprintf("DELRR%d", delrridx)
		oldRecordString := chng.Existing.Original.(string)
		params[key] = oldRecordString
		fmt.Fprintf(&builder, "\033[31m- %s = %s\033[0m\n", key, oldRecordString)
		delrridx++
		// new record creation
		newRecordString, err := n.createRecordString(chng.Desired, dc.Name)
		if err != nil {
			return corrections, 0, err
		}
		key = fmt.Sprintf("ADDRR%d", addrridx)
		params[key] = newRecordString
		fmt.Fprintf(&builder, "\033[32m+ %s = %s\033[0m\n", key, newRecordString)
		addrridx++
	}

	if changes {
		msg := fmt.Sprintf("GENERATE_ZONE: %s\n%s", dc.Name, buf.String())
		if n.isDebugOn() {
			msg = fmt.Sprintf("GENERATE_ZONE: %s\n%sPROVIDER CNR, API COMMAND PARAMETERS:\n%s", dc.Name, buf.String(), builder.String())
		}
		corrections = append(corrections, &models.Correction{
			Msg: msg,
			F: func() error {
				return n.updateZoneBy(params, dc.Name)
			},
		})
	}

	dnssecCorrections, err := n.getDNSSECCorrections(dc)
	if err != nil {
		return nil, 0, err
	}
	corrections = append(corrections, dnssecCorrections...)
	actualChangeCount += len(dnssecCorrections)

	return corrections, actualChangeCount, nil
}

func toRC(dc *models.DomainConfig, data map[string]string) (*models.RecordConfig, error) {

	ttl, err := strconv.ParseUint(data["TTL"], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid TTL value for domain %s: %s", dc.Name, data["TTL"])
	}

	rc, err := dc.NewRecordConfigParse(dc.LabelFromShort(data["NAME"]), uint32(ttl), data["TYPE"], data["CONTENT"], nrc.TARGET_IS_FQDN_NO_DOT)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	rc.Original = deleteRecordString(rc) // This is the code we'll need to delete the record.

	return rc, nil
}

// updateZoneBy updates the zone with the provided changes.
func (n *Client) updateZoneBy(params map[string]any, domain string) error {
	zone := domain
	cmd := map[string]any{
		"COMMAND": "ModifyDNSZone",
		"DNSZONE": zone,
	}
	maps.Copy(cmd, params)
	r := n.client.Request(cmd)
	if !r.IsSuccess() {
		return n.GetAPIError("Error while updating zone", zone, r)
	}
	return nil
}

// getRecords queries the API for all resource records of a zone.
func (n *Client) getRecords(dc *models.DomainConfig) (models.Records, error) {
	var records models.Records
	domain := dc.Name

	// Command to find out the total numbers of resource records for the zone
	// so that the follow-up query can be done with the correct limit
	cmd := map[string]any{
		"COMMAND": "QueryDNSZoneRRList",
		"DNSZONE": domain,
		"ORDERBY": "type",
		"FIRST":   "0",
		"LIMIT":   "10000",
		"WIDE":    "1",
	}
	r := n.client.Request(cmd)

	// Check if the request was successful
	if !r.IsSuccess() {
		if r.GetCode() == 545 {
			// If dns zone does not exist create a new one automatically
			if !isNoPopulate() {
				err := n.EnsureZoneExists(dc)
				if err != nil {
					return nil, err
				}
			} else {
				// Return specific error if the zone does not exist
				return nil, n.GetAPIError("Use `dnscontrol create-domains` to create not-existing zone", domain, r)
			}
		}
		// Return general error for any other issues
		return nil, n.GetAPIError("Failed loading resource records for zone", domain, r)
	}
	totalRecords := r.GetRecordsTotalCount()
	if totalRecords <= 0 {
		return nil, nil
	}

	// loop over the records array
	rrs := r.GetRecords()
	for i := range len(rrs) {
		data := rrs[i].GetData()
		if _, exists := data["NAME"]; !exists {
			continue
		}

		record, err := toRC(dc, data)
		if err != nil {
			return nil, fmt.Errorf("toRC error: %w", err)
		}
		records = append(records, record)
	}

	// Return the slice of records
	return records, nil
}

// Function to create record string from given RecordConfig for the ADDRR# API parameter.
func (n *Client) createRecordString(rc *models.RecordConfig, domain string) (string, error) {

	host := rc.GetLabel()
	// Apex records are represented by domain+".".
	if host == domain {
		host += "."
	}

	var answer string

	switch rc.Type { // #rtype_variations
	case "LOC":
		// Use .String() returns the properly formatted LOC string
		// via the dns library (e.g. "52 14 5.000 N 000 08 50.000 E 10.00m 0.00m 0.00m 0.00m")
		parts := strings.Fields(rc.GetRDATA().String())
		altitude, _ := strconv.ParseFloat(strings.TrimSuffix(parts[8], "m"), 64)
		size, _ := strconv.ParseFloat(strings.TrimSuffix(parts[9], "m"), 64)
		hp, _ := strconv.ParseFloat(strings.TrimSuffix(parts[10], "m"), 64)
		vp, _ := strconv.ParseFloat(strings.TrimSuffix(parts[11], "m"), 64)
		answer = fmt.Sprintf("%s %s %s %s %s %s %s %s %.2fm %.2fm %.2fm %.2fm",
			parts[0], parts[1], parts[2], parts[3],
			parts[4], parts[5], parts[6], parts[7],
			altitude, size, hp, vp)
	case "SVCB", "HTTPS":
		answer = rc.GetRDATA().String()
		answer = strings.ReplaceAll(answer, `"`, ``)
	case "SSHFP":
		f := rc.AsSSHFP()
		answer = fmt.Sprintf(`%v %v %s`, f.Algorithm, f.Type, f.FingerPrint)
	case "NAPTR":
		f := rc.AsNAPTR()
		answer = fmt.Sprintf(`%v %v "%v" "%v" "%v" %v`, f.Order, f.Preference, f.Flags, f.Service, f.Regexp, f.Replacement)
	case "TLSA":
		f := rc.AsTLSA()
		answer = fmt.Sprintf(`%v %v %v %s`, f.Usage, f.Selector, f.MatchingType, f.Certificate)
	case "SMIMEA":
		f := rc.AsSMIMEA()
		answer = fmt.Sprintf(`%v %v %v %s`, f.Usage, f.Selector, f.MatchingType, f.Certificate)
	case "CAA":
		f := rc.AsCAA()
		answer = fmt.Sprintf(`%v %s "%s"`, f.Flag, f.Tag, f.Value)
	default:
		answer = rc.GetRDATA().String()
	}

	var ifIn string
	if rc.Type != "NS" {
		ifIn = " IN"
	}

	return fmt.Sprintf("%s %d%s %s %s", host, rc.TTL, ifIn, rc.Type, answer), nil
}

// deleteRecordString constructs the record string based on the provided Record.
func deleteRecordString(rc *models.RecordConfig) string {
	switch rc.Type {
	case "MX":
		return fmt.Sprintf("%s %d IN MX %s", rc.GetLabel(), rc.TTL, rc.AsMX().Mx)
	case "NS":
		return fmt.Sprintf("%s %d NS %s", rc.GetLabel(), rc.TTL, rc.AsNS().Ns)
	case "SVCB", "HTTPS":
		d := rc.GetRDATA().String()
		d = strings.ReplaceAll(d, `"`, ``)
		return fmt.Sprintf("%s %d IN %s %s", rc.GetLabel(), rc.TTL, rc.Type, d)
	case "TLSA":
		d := rc.GetRDATA().String()
		d = strings.ToLower(d)
		return fmt.Sprintf("%s %d IN %s %s", rc.GetLabel(), rc.TTL, rc.Type, d)
	}
	return fmt.Sprintf("%s %d IN %s %s", rc.GetLabel(), rc.TTL, rc.Type, rc.GetRDATA().String())
}

// Function to check the no-populate argument.
func isNoPopulate() bool {
	return slices.Contains(os.Args, "--no-populate")
}

// Function to check if debug mode is enabled.
func (n *Client) isDebugOn() bool {
	debugMode, exists := n.conf["debugmode"]
	return exists && (debugMode == "1" || debugMode == "2")
}

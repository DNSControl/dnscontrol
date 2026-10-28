package primedns

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// skipRecord reports whether a record of the zone is invisible to DNSControl.
//
//   - SOA is managed by PrimeDNS.
//   - The apex NS records that point at PrimeDNS's own name servers are managed by PrimeDNS. Other apex NS records
//     (a second provider for dual hosting) are kept.
//   - Rows with an empty type are empty non-terminals, not records.
//   - CDS, CDNSKEY and URI exist at PrimeDNS but DNSControl has no such record types.
func skipRecord(domain string, r record) bool {
	switch r.Type {
	case "":
		// An empty non-terminal: the API lists the helper row of a name that only has names below it
		// (_domainkey when default._domainkey exists) with an empty type, TTL 0 and empty content. It is not a record.
		return true
	case "SOA", "CDS", "CDNSKEY", "URI":
		return true
	case "NS":
		if strings.EqualFold(strings.TrimSuffix(r.Name, "."), domain) {
			target := strings.ToLower(strings.TrimSuffix(r.Content, "."))
			return strings.HasSuffix(target, ".no-ip.pl") || strings.HasSuffix(target, ".primedns.net")
		}
	}
	return false
}

// noDot removes the trailing dot of a domain name, but leaves the root name "." (the null target of SRV) alone.
func noDot(s string) string {
	if s == "." {
		return s
	}
	return strings.TrimSuffix(s, ".")
}

// withDot makes sure a domain name ends with a dot (the API stores names the way they were sent).
func withDot(s string) string {
	if s == "" || strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

// txtToPresentation turns TXT content as the API stores it into zone file presentation format. PrimeDNS stores
// TXT as free text and PowerDNS puts the quotes around it when the text does not start with one, so
// part1" "part2 is two strings and "part1" "part2" is the same two strings.
func txtToPresentation(content string) string {
	if strings.HasPrefix(content, `"`) {
		return content
	}
	return `"` + content + `"`
}

// toRecordConfig converts a PrimeDNS record to a RecordConfig.
func toRecordConfig(dc *models.DomainConfig, r record) (*models.RecordConfig, error) {
	label := dc.LabelFromFQDNNoDot(strings.TrimSuffix(r.Name, "."))
	ttl := uint32(r.TTL)
	content := r.Content

	var rc *models.RecordConfig
	var err error
	switch r.Type {
	case "A", "AAAA":
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, content)
	case "CNAME", "NS", "PTR", "ALIAS", "DNAME":
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, withDot(content))
	case "MX":
		prio, target, perr := splitPrio(r.Prio, content)
		if perr != nil {
			return nil, perr
		}
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, uint16(prio), withDot(target))
	case "SRV":
		// the API keeps the priority in its own field and the content as "weight port target"
		fields := strings.Fields(content)
		if len(fields) != 3 || r.Prio == nil {
			return nil, fmt.Errorf("unexpected SRV record: prio %v content %q", r.Prio, content)
		}
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, fmt.Sprintf("%d %s %s %s", *r.Prio, fields[0], fields[1], withDot(fields[2])))
	case "TXT":
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, txtToPresentation(content))
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, content)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r
	return rc, nil
}

// splitPrio returns the priority of an MX record: the API field, or the leading number of the content.
func splitPrio(prio *int, content string) (int, string, error) {
	if prio != nil {
		return *prio, content, nil
	}
	parts := strings.SplitN(strings.TrimSpace(content), " ", 2)
	if len(parts) == 2 {
		if n, err := strconv.Atoi(parts[0]); err == nil {
			return n, strings.TrimSpace(parts[1]), nil
		}
	}
	return 0, "", fmt.Errorf("MX record without a priority: %q", content)
}

// fromRecordConfig converts a RecordConfig to the body of a create or update request.
func fromRecordConfig(rc *models.RecordConfig) recordInput {
	in := recordInput{
		Name: rc.GetLabel(), // "@" for the apex, the short label otherwise
		Type: rc.Type,
		TTL:  int(rc.TTL),
	}
	rdata := rc.GetRDATA().String()
	switch rc.Type {
	case "CNAME", "NS", "ALIAS", "DNAME":
		in.Content = noDot(rdata)
	case "PTR":
		in.Content = withDot(rdata) // the API wants a trailing dot here
	case "MX":
		f := rc.AsMX()
		prio := int(f.Preference)
		in.Prio = &prio
		in.Content = noDot(f.Mx)
	case "SRV":
		// "priority weight port target": the priority goes into its own field
		fields := strings.Fields(rdata)
		if len(fields) == 4 {
			prio, _ := strconv.Atoi(fields[0])
			in.Prio = &prio
			in.Content = fmt.Sprintf("%s %s %s", fields[1], fields[2], noDot(fields[3]))
		} else {
			in.Content = rdata
		}
	case "HTTPS", "SVCB":
		in.Content = stripSvcbQuotes(rdata)
	default: // A, AAAA, TXT (already quoted), CAA, SSHFP, TLSA, DS, NAPTR, LOC, OPENPGPKEY ...
		in.Content = rdata
	}
	return in
}

// stripSvcbQuotes removes the quotes around SvcParam values ("alpn=h2,h3" is what the API stores) and keeps
// them around ech, whose base64 value needs them.
func stripSvcbQuotes(s string) string {
	const ech = `ech="`
	i := strings.Index(s, ech)
	if i < 0 {
		return strings.ReplaceAll(s, `"`, "")
	}
	end := strings.Index(s[i+len(ech):], `"`)
	if end < 0 {
		return strings.ReplaceAll(s, `"`, "")
	}
	end += i + len(ech)
	return strings.ReplaceAll(s[:i], `"`, "") + s[i:end+1] + strings.ReplaceAll(s[end+1:], `"`, "")
}

package gigahost

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

// features describes the capabilities of the Gigahost provider. Start with
// zero optional capabilities; only A/AAAA/CNAME/MX/TXT/NS are handled.
var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones: providers.Can(),
	providers.CanConcur:   providers.Unimplemented(),
	providers.CanUseAlias: providers.Can(),
	providers.CanUseCAA:   providers.Can(),
	providers.CanUseDNAME: providers.Can(),
	providers.CanUseNAPTR: providers.Can(),
	providers.CanUsePTR:   providers.Can(),
	providers.CanUseSRV:   providers.Can(),
}

func init() {
	const providerName = "GIGAHOST"
	const providerMaintainer = "@jochristian"
	fns := providers.DspFuncs{
		Initializer:   newGigahost,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Gigahost",
		Kind:        providers.KindDNS,
		Fields: []providers.CredsField{
			{
				Key:      "apikey",
				Label:    "API key",
				Help:     "Gigahost API key with DNS read-write permission (flux_live_...).",
				Secret:   true,
				Required: true,
			},
		},
	})
}

type gigahostProvider struct {
	apiKey string
	zones  map[string]zone // zone_name -> zone
}

func newGigahost(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	apiKey := settings["apikey"]
	if apiKey == "" {
		return nil, errors.New("gigahost: missing 'apikey' in creds.json")
	}
	return &gigahostProvider{apiKey: apiKey}, nil
}

// AuditRecords returns a list of errors corresponding to the records that
// aren't supported by this provider.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}
	a.Add("MX", rejectif.MxNull) // The API rejects a "." target ("MX record value must be a valid mail server hostname").
	a.Add("TXT", rejectif.TxtIsEmpty)
	// The API cannot round-trip double quotes in TXT values: raw interior
	// quotes are rejected by its zone parser, and if sent escaped
	// (`"right\""`) the record is stored correctly but the read-back
	// record_value is mangled (`right\`), breaking diffing and deletes.
	a.Add("TXT", rejectif.TxtHasDoubleQuotes)
	a.Add("TXT", rejectif.TxtStartsOrEndsWithSpaces) // The API trims leading/trailing whitespace on store.
	a.Add("CAA", rejectif.CaaTargetContainsWhitespace)
	a.Add("SRV", rejectif.SrvHasNullTarget)
	errs := a.Audit(records)

	// The API silently replaces the entire NAPTR RRset on every create
	// (POSTing a second NAPTR at a label deletes the first), so only one
	// NAPTR per label can be represented. Verified against the live API
	// 2026-07-27.
	naptrSeen := map[string]bool{}
	for _, rc := range records {
		if rc.Type != "NAPTR" {
			continue
		}
		if naptrSeen[rc.GetLabelFQDN()] {
			errs = append(errs, fmt.Errorf("only one NAPTR record per label is supported (%s)", rc.GetLabelFQDN()))
		}
		naptrSeen[rc.GetLabelFQDN()] = true
	}
	return errs
}

// findZone resolves a domain name to its Gigahost zone, caching the zone list.
func (c *gigahostProvider) findZone(domain string) (zone, error) {
	if c.zones == nil {
		zones, err := c.getAllZones()
		if err != nil {
			return zone{}, err
		}
		c.zones = make(map[string]zone, len(zones))
		for _, z := range zones {
			c.zones[z.ZoneName] = z
		}
	}
	z, ok := c.zones[domain]
	if !ok {
		return zone{}, fmt.Errorf("gigahost: %q is not a zone in this account", domain)
	}
	return z, nil
}

// ListZones returns all zone names in the account.
func (c *gigahostProvider) ListZones() ([]string, error) {
	zones, err := c.getAllZones()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(zones))
	for _, z := range zones {
		names = append(names, z.ZoneName)
	}
	return names, nil
}

// gigahostNameservers are Gigahost's authoritative nameservers. Gigahost has no
// per-zone parent-nameserver endpoint; every zone it hosts is served by this
// fixed set. Confirmed by the provider 2026-06-23.
var gigahostNameservers = []string{
	"ns1.gigahost.no",
	"ns2.gigahost.no",
	"ns3.gigahost.no",
}

// GetNameservers returns Gigahost's authoritative nameservers for a domain.
func (c *gigahostProvider) GetNameservers(_ string) ([]*models.Nameserver, error) {
	return models.ToNameservers(gigahostNameservers)
}

// GetZoneRecords gets all records of a zone in RecordConfig format.
func (c *gigahostProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	z, err := c.findZone(dc.Name)
	if err != nil {
		return nil, err
	}
	recs, err := c.getRecords(z.ZoneID)
	if err != nil {
		return nil, err
	}
	result := make(models.Records, 0, len(recs))
	for i := range recs {
		t := recs[i].RecordType
		// SOA is managed by Gigahost and not exposed as a writable record;
		// silently ignore it so it is never proposed for deletion.
		if t == "SOA" {
			continue
		}
		// Leave record types we don't support untouched rather than
		// clobbering them: ignore them on read with a warning.
		if !supportedTypes[t] {
			printer.Warnf("GIGAHOST: ignoring unsupported record type %s (%s)\n", t, recs[i].RecordName)
			continue
		}
		rc, err := nativeToRecordConfig(dc, &recs[i])
		if err != nil {
			return nil, err
		}
		result = append(result, rc)
	}
	return result, nil
}

// supportedTypes is the set of record types this provider manages.
var supportedTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true, "MX": true, "TXT": true, "NS": true,
	"ALIAS": true, "CAA": true, "DNAME": true, "NAPTR": true, "PTR": true, "SRV": true,
}

// GetZoneRecordsCorrections computes the changes needed to bring the zone in
// sync with dc, using the per-record diff strategy.
func (c *gigahostProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	z, err := c.findZone(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	instructions, actualChangeCount, err := diff2.ByRecord(existing, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, inst := range instructions {
		switch inst.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: inst.MsgsJoined})
		case diff2.CREATE:
			newRec := inst.New[0]
			corrections = append(corrections, &models.Correction{
				Msg: inst.Msgs[0],
				F: func() error {
					return c.createRecord(z.ZoneID, recordConfigToRequest(newRec))
				},
			})
		case diff2.CHANGE:
			oldRec := inst.Old[0]
			newRec := inst.New[0]
			id := oldRec.Original.(*record).RecordID
			corrections = append(corrections, &models.Correction{
				Msg: inst.Msgs[0],
				F: func() error {
					return c.updateRecord(z.ZoneID, id, recordConfigToRequest(newRec))
				},
			})
		case diff2.DELETE:
			old := inst.Old[0].Original.(*record)
			corrections = append(corrections, &models.Correction{
				Msg: inst.Msgs[0],
				F: func() error {
					return c.deleteRecord(z.ZoneID, old.RecordID, old.RecordName, old.RecordType, deleteValue(old))
				},
			})
		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

// nativeToRecordConfig converts a Gigahost record into a RecordConfig.
func nativeToRecordConfig(dc *models.DomainConfig, r *record) (*models.RecordConfig, error) {
	var rc *models.RecordConfig
	var err error
	switch r.RecordType {
	case "MX":
		rc, err = dc.NewRecordConfig(r.RecordName, r.RecordTTL.Value, r.RecordType, r.RecordPrio.Value, addDot(r.RecordValue))
	case "CNAME", "NS", "ALIAS", "PTR", "DNAME":
		// Gigahost stores hostname targets inconsistently (some with a trailing
		// dot, some without); RecordConfig targets are always FQDNs.
		rc, err = dc.NewRecordConfig(r.RecordName, r.RecordTTL.Value, r.RecordType, addDot(r.RecordValue))
	case "TXT":
		val := r.RecordValue
		// The API returns >255-octet TXT values in the RFC1035 chunked form
		// we sent (see recordConfigToRequest) with the outer quotes stripped,
		// e.g. `aaa...aaa" "bbb`. A single TXT character-string cannot exceed
		// 255 octets, so a longer record_value must be that form: re-wrap and
		// decode to the raw joined text. Shorter values are stored and
		// returned verbatim (kept as-is on decode failure).
		if len(val) > 255 {
			if decoded, derr := txtutil.ParseQuoted(`"` + val + `"`); derr == nil {
				val = decoded
			}
		}
		rc, err = dc.NewRecordConfig(r.RecordName, r.RecordTTL.Value, r.RecordType, val)
	default:
		rc, err = dc.NewRecordConfigParse(r.RecordName, r.RecordTTL.Value, r.RecordType, r.RecordValue)
	}
	if err != nil {
		return nil, fmt.Errorf("gigahost: unparsable %s record %q: %w", r.RecordType, r.RecordValue, err)
	}
	rc.Original = r
	return rc, nil
}

// recordConfigToRequest converts a RecordConfig into a Gigahost request body.
func recordConfigToRequest(rc *models.RecordConfig) *recordRequest {
	r := &recordRequest{
		RecordName: rc.GetLabel(),
		RecordType: rc.Type,
		RecordTTL:  rc.TTL,
	}
	switch rc.Type {
	case "MX":
		pref := rc.MxPreference
		r.RecordPrio = &pref
		r.RecordValue = strings.TrimSuffix(rc.GetTargetField(), ".")
	case "CNAME", "NS", "ALIAS", "PTR", "DNAME":
		// Send hostname targets without a trailing dot; the API normalizes as
		// needed and reads are re-dotted in nativeToRecordConfig.
		r.RecordValue = strings.TrimSuffix(rc.GetTargetField(), ".")
	case "TXT":
		txt := rc.GetTargetTXTJoined()
		// The API requires values over 255 octets to be sent as RFC1035
		// quoted 255-octet chunks: `"aaa...aaa" "bbb"`. Shorter values are
		// sent raw and stored verbatim. (Values with double quotes are
		// rejected by AuditRecords: the API cannot round-trip them.)
		if len(txt) > 255 {
			txt = txtutil.EncodeQuoted(txt)
		}
		r.RecordValue = txt
	case "CAA", "SRV", "NAPTR":
		// These are stored as full RFC1035 presentation strings in record_value.
		r.RecordValue = rc.GetRDATA().String()
	default:
		r.RecordValue = rc.GetTargetField()
	}
	return r
}

// deleteValue returns the value the DELETE endpoint matches against. For most
// types this is record_value exactly as the API returns it, but for MX the
// endpoint matches the internal RDATA presentation ("<priority> <target.>"),
// NOT the priority-less record_value it returns; sending record_value yields
// 404 "Record not found". Verified against the live API 2026-07-27.
func deleteValue(r *record) string {
	if r.RecordType == "MX" {
		return fmt.Sprintf("%d %s", r.RecordPrio.Value, addDot(r.RecordValue))
	}
	return r.RecordValue
}

// addDot appends a trailing dot to a hostname if it lacks one.
func addDot(s string) string {
	if s == "" || strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

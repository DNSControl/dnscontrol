package nexdns

import (
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

// apexLabel is how both DNSControl and the API name the zone apex.
const apexLabel = "@"

// toRecordConfig converts one record of an API response to a RecordConfig.
// A response carries the assembled rdata in Content ("10 mail.example.com." for
// an MX), which is what NewRecordConfigParse parses, so no per-type handling is
// needed on the way in - except for TXT.
//
// TXT is asymmetric: the API accepts a value verbatim but returns it in RFC 1035
// presentation form, quoted and escaped, and chunked into 255-octet strings when
// needed.
func toRecordConfig(dc *models.DomainConfig, r apiRecord) (*models.RecordConfig, error) {
	rc, err := dc.NewRecordConfigParse(dc.LabelFromShort(r.Name), uint32(r.TTL), r.Type, r.Content,
		nrc.Flags{})
	if err != nil {
		return nil, err
	}

	rc.Original = r

	return rc, nil
}

// fromRecordConfig converts a RecordConfig to the body of a create or update.
// A request is not shaped like a response: Content carries only the primary
// value, and the numeric and keyword parts of a type travel as their own fields,
// so the assembled rdata is taken apart again here.
//
// Hostname targets are sent as the FQDN that DNSControl holds, trailing dot and
// all, which is the form the API stores them in.
func fromRecordConfig(rc *models.RecordConfig) recordRequest {
	req := recordRequest{
		Name: rc.GetLabel(),
		Type: rc.Type,
		TTL:  int(rc.TTL),
	}

	switch f := rc.GetRDATA().(type) {
	case dnsrdatav2.MX:
		req.Priority = new(int(f.Preference))
		req.Content = f.Mx
	case dnsrdatav2.SRV:
		req.Priority = new(int(f.Priority))
		req.Weight = new(int(f.Weight))
		req.Port = new(int(f.Port))
		req.Content = f.Target
	case dnsrdatav2.CAA:
		req.Flags = new(int(f.Flag))
		req.Tag = f.Tag
		req.Content = f.Value
	case dnsrdatav2.DS:
		req.KeyTag = new(int(f.KeyTag))
		req.Algorithm = new(int(f.Algorithm))
		req.DigestType = new(int(f.DigestType))
		req.Content = f.Digest
	case dnsrdatav2.TLSA:
		// The certificate association data is the target; the three selectors
		// are separate fields in the model as well as in the request.
		req.Usage = new(int(f.Usage))
		req.Selector = new(int(f.Selector))
		req.MatchingType = new(int(f.MatchingType))
		req.Content = f.Certificate
	case dnsrdatav2.TXT:
		req.Content = rc.GetTargetTXTJoined()
	default:
		req.Content = rc.GetRDATA().String()
	}

	return req
}

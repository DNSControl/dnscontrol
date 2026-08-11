package nexdns

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

// apexLabel is how both DNSControl and the API name the zone apex.
const apexLabel = "@"

// toRecordConfig converts one record of an API response to a RecordConfig.
// A response carries the assembled rdata in Content ("10 mail.example.com." for
// an MX), which is what PopulateFromString parses, so no per-type handling is
// needed on the way in - except for TXT.
//
// TXT is asymmetric: the API accepts a value verbatim but returns it in RFC 1035
// presentation form, quoted and escaped, and chunked into 255-octet strings when
// it is longer than that. PopulateFromString cannot undo that - it only strips
// the outer quotes - so a backslash returned as `\\` would be read back as two
// characters, written again as four, and doubled on every run until the record
// no longer says what its owner wrote. txtutil.ParseQuoted is the decoder for
// this form, and it passes an unquoted value through unchanged.
func toRecordConfig(r apiRecord, origin string) (*models.RecordConfig, error) {
	rc := &models.RecordConfig{
		Type:     r.Type,
		TTL:      uint32(r.TTL),
		Original: r,
	}
	rc.SetLabel(r.Name, origin)

	if r.Type == "TXT" {
		value, err := txtutil.ParseQuoted(r.Content)
		if err != nil {
			return nil, fmt.Errorf("unparsable TXT value %q: %w", r.Content, err)
		}
		if err := rc.SetTargetTXT(value); err != nil {
			return nil, err
		}

		return rc, nil
	}

	if err := rc.PopulateFromString(r.Type, r.Content, origin); err != nil {
		return nil, err
	}

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

	switch rc.Type {
	case "MX":
		req.Content = rc.GetTargetField()
		req.Priority = new(int(rc.MxPreference))
	case "SRV":
		req.Content = rc.GetTargetField()
		req.Priority = new(int(rc.SrvPriority))
		req.Weight = new(int(rc.SrvWeight))
		req.Port = new(int(rc.SrvPort))
	case "CAA":
		req.Content = rc.GetTargetField()
		req.Flags = new(int(rc.CaaFlag))
		req.Tag = rc.CaaTag
	case "DS":
		req.Content = rc.DsDigest
		req.KeyTag = new(int(rc.DsKeyTag))
		req.Algorithm = new(int(rc.DsAlgorithm))
		req.DigestType = new(int(rc.DsDigestType))
	case "TLSA":
		// The certificate association data is the target; the three selectors
		// are separate fields in the model as well as in the request.
		req.Content = rc.GetTargetField()
		req.Usage = new(int(rc.TlsaUsage))
		req.Selector = new(int(rc.TlsaSelector))
		req.MatchingType = new(int(rc.TlsaMatchingType))
	case "TXT":
		req.Content = rc.GetTargetTXTJoined()
	default:
		req.Content = rc.GetTargetField()
	}

	return req
}

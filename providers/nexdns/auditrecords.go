package nexdns

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// caaTags are the property tags the API accepts in a CAA record.
var caaTags = []string{"issue", "issuewild", "iodef"}

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider.  If all records are
// supported, an empty list is returned.
func AuditRecords(records []*models.RecordConfig) []error {
	a := rejectif.Auditor{}

	// validation_error - value: Value is required. Last verified 2026-07-31.
	a.Add("TXT", rejectif.TxtIsEmpty)

	// The API trims a TXT value before storing it, so a leading or trailing
	// space is dropped without being reported. Left alone that reads as a
	// permanently pending change - the record never matches what was asked for,
	// and every run offers the same correction. Refusing it names the reason
	// once, before anything is written. Interior whitespace is kept and is
	// therefore not rejected. Last verified 2026-07-31.
	a.Add("TXT", rejectif.TxtHasTrailingSpace)
	a.Add("TXT", rejectTxtLeadingSpace)

	// The platform serves public zones only and turns away an address that
	// cannot be reached from the internet, which is also what stops a zone from
	// being used to point a public name at a visitor's own network. Catching it
	// here names the address before a push starts, instead of failing partway
	// through with some records already written. Last verified 2026-07-31.
	a.Add("A", rejectUnroutableIP)
	a.Add("AAAA", rejectUnroutableIP)

	// The CAA value is stored inside quotes, so whitespace in it would produce
	// rdata the nameserver refuses. The API turns that away first, with
	// validation_error - value: CAA value must not contain spaces or quotes.
	// Last verified 2026-07-31.
	a.Add("CAA", rejectif.CaaTargetContainsWhitespace)

	// Only the three property tags of RFC 8659 are accepted; anything else is
	// validation_error - tag: Tag must be issue, issuewild, or iodef.
	// Last verified 2026-07-31.
	a.Add("CAA", rejectCaaTag)

	return a.Audit(records)
}

// thisNetwork and reservedIPv4 are the two refused ranges that the netip
// predicates below do not already cover. 240.0.0.0/4 takes in the broadcast
// address with it.
var (
	thisNetwork  = netip.MustParsePrefix("0.0.0.0/8")
	reservedIPv4 = netip.MustParsePrefix("240.0.0.0/4")
)

// rejectUnroutableIP rejects A and AAAA records whose address the API refuses.
//
// The set was read off the API rather than off a list of well-known ranges,
// because the two do not agree: carrier-grade NAT (100.64.0.0/10),
// documentation ranges (192.0.2.0/24, 2001:db8::/32) and multicast are all
// accepted, so rejecting them here would block a record the platform is
// perfectly willing to store.
func rejectUnroutableIP(rc *models.RecordConfig) error {
	ip := rc.GetTargetIP()
	if !ip.IsValid() {
		return nil // Not an address at all; a different check reports that.
	}

	unroutable := ip.IsPrivate() || // RFC 1918 and fc00::/7
		ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsUnspecified() ||
		ip.Is4In6() ||
		thisNetwork.Contains(ip) ||
		reservedIPv4.Contains(ip)

	if unroutable {
		return fmt.Errorf("%s is a private or reserved address", ip)
	}

	return nil
}

// rejectTxtLeadingSpace rejects TXT records whose value starts with a space.
// rejectif has a helper for the trailing case but not for this one, because the
// integration suite leaves leading whitespace untested - it is nonetheless
// trimmed the same way here.
func rejectTxtLeadingSpace(rc *models.RecordConfig) error {
	if strings.HasPrefix(rc.GetTargetTXTJoined(), " ") {
		return errors.New("txtstring starts with space")
	}
	return nil
}

// rejectCaaTag rejects CAA records whose property tag the API does not accept.
func rejectCaaTag(rc *models.RecordConfig) error {
	if !slices.Contains(caaTags, rc.AsCAA().Tag) {
		return fmt.Errorf("caa tag %q is not supported", rc.AsCAA().Tag)
	}
	return nil
}

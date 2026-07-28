package powerdns

import (
	"fmt"
	"strconv"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	svcbv2 "codeberg.org/miekg/dns/svcb"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/mittwald/go-powerdns/apis/zones"
)

// toRecordConfig converts a PowerDNS DNSRecord to a RecordConfig. #rtype_variations.
func toRecordConfig(dc *models.DomainConfig, r zones.Record, ttl int, name string, rtype string) (*models.RecordConfig, error) {
	label := dc.LabelFromFQDNWithDot(name)
	var rc *models.RecordConfig
	var err error
	switch rtype {
	case "TXT":
		// PowerDNS API accepts long TXTs without requiring to split them.
		// The API then returns them as they initially came in, e.g. "averylooooooo[...]oooooongstring" or "string" "string"
		// So we need to strip away " and split into multiple string
		// We can't use SetTargetRFC1035Quoted, it would split the long strings into multiple parts
		rc, err = dc.NewRecordConfig(label, uint32(ttl), dnsv2.TypeTXT, strings.Join(parseTxt(r.Content), ""))
	case "LUA":
		luaType, payload := models.ParseLuaContent(r.Content)
		var value string
		value, err = models.DecodeLuaPayload(payload)
		if err != nil {
			return nil, err
		}
		rc, err = dc.NewRecordConfig(label, uint32(ttl), privatetypes.TypeLUA, luaType, value)
	case "HTTPS", "SVCB":
		if contentHasPowerDNSSVCBAutoHints(r.Content) {
			rc, err = newPowerDNSSVCBAutoHintRecord(dc, label, uint32(ttl), rtype, r.Content)
		} else {
			rc, err = dc.NewRecordConfigParse(label, uint32(ttl), rtype, r.Content)
		}
	default:
		rc, err = dc.NewRecordConfigParse(label, uint32(ttl), rtype, r.Content)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r
	return rc, nil
}

func newPowerDNSSVCBAutoHintRecord(dc *models.DomainConfig, label string, ttl uint32, rtype, content string) (*models.RecordConfig, error) {
	fields := strings.Fields(content)
	if len(fields) < 2 {
		return nil, fmt.Errorf("could not parse PowerDNS SVCB record: %s", content)
	}
	priority, err := strconv.ParseUint(fields[0], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("could not parse PowerDNS SVCB priority %q: %w", fields[0], err)
	}
	rc, err := dc.NewRecordConfig(label, ttl, rtype, uint16(priority), fields[1], []svcbv2.Pair{})
	if err != nil {
		return nil, err
	}
	rc.SvcParams = strings.Join(fields[2:], " ")
	return rc, nil
}

func parseTxt(content string) (result []string) {
	for r := range strings.SplitSeq(content, "\" ") {
		result = append(result, strings.Trim(r, "\""))
	}
	return
}

package powerdns

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/mittwald/go-powerdns/apis/zones"
)

// toRecordConfig converts a PowerDNS DNSRecord to a RecordConfig. #rtype_variations.
func toRecordConfig(domain string, r zones.Record, ttl int, name string, rtype string) (*models.RecordConfig, error) {
	// trimming trailing dot and domain from name
	name = strings.TrimSuffix(name, domain+".")
	name = strings.TrimSuffix(name, ".")

	rc := &models.RecordConfig{
		TTL:      uint32(ttl),
		Original: r,
		Type:     rtype,
	}
	rc.SetLabel(name, domain)

	switch rtype {
	case "TXT":
		// PowerDNS API accepts long TXTs without requiring to split them.
		// The API then returns them as they initially came in, e.g. "averylooooooo[...]oooooongstring" or "string" "string"
		// So we need to strip away " and split into multiple string
		// We can't use SetTargetRFC1035Quoted, it would split the long strings into multiple parts
		return rc, rc.SetTargetTXTs(parseTxt(r.Content))
	case "LUA":
		luaType, payload := models.ParseLuaContent(r.Content)
		rc.LuaRType = luaType
		value, err := models.DecodeLuaPayload(payload)
		if err != nil {
			return nil, err
		}
		return rc, rc.SetTargetTXT(value)
	case "HTTPS", "SVCB":
		if contentHasPowerDNSSVCBAutoHints(r.Content) {
			return rc, setTargetSVCBPowerDNS(rc, r.Content)
		}
		return rc, rc.PopulateFromString(rtype, r.Content, domain)
	default:
		return rc, rc.PopulateFromString(rtype, r.Content, domain)
	}
}

func setTargetSVCBPowerDNS(rc *models.RecordConfig, content string) error {
	fields := strings.Fields(content)
	if len(fields) < 2 {
		return fmt.Errorf("could not parse PowerDNS SVCB record: %s", content)
	}
	priority, err := strconv.ParseUint(fields[0], 10, 16)
	if err != nil {
		return fmt.Errorf("could not parse PowerDNS SVCB priority %q: %w", fields[0], err)
	}
	rc.SvcPriority = uint16(priority)
	if err := rc.SetTarget(fields[1]); err != nil {
		return err
	}
	rc.SvcParams = strings.Join(fields[2:], " ")
	return nil
}

func parseTxt(content string) (result []string) {
	for r := range strings.SplitSeq(content, "\" ") {
		result = append(result, strings.Trim(r, "\""))
	}
	return
}

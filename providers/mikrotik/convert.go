package mikrotik

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
)

// nativeToRecords converts a RouterOS DNS static record to dnscontrol RecordConfig(s).
func nativeToRecords(nr dnsStaticRecord, dc *models.DomainConfig) (models.Records, error) {
	ttl, err := parseMikrotikDuration(nr.TTL)
	if err != nil {
		return nil, fmt.Errorf("invalid TTL %q: %w", nr.TTL, err)
	}
	label := dc.LabelFromFQDNNoDot(nr.Name)
	var rc *models.RecordConfig

	switch nr.Type {
	case "A":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeA, nr.Address)

	case "AAAA":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeAAAA, nr.Address)

	case "CNAME":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeCNAME, ensureTrailingDot(nr.CName))

	case "FWD":
		rc, err = dc.NewRecordConfig(label, ttl, privatetypes.TypeMIKROTIKFWD, nr.ForwardTo)

	case "NXDOMAIN":
		rc, err = dc.NewRecordConfig(label, ttl, privatetypes.TypeMIKROTIKNXDOMAIN)

	case "MX":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, nr.MxPreference, ensureTrailingDot(nr.MxExchange))

	case "NS":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeNS, ensureTrailingDot(nr.NS))

	case "SRV":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeSRV, nr.SrvPriority, nr.SrvWeight, nr.SrvPort, ensureTrailingDot(nr.SrvTarget))

	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, nr.Text)

	default:
		return nil, fmt.Errorf("unsupported record type %q", nr.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid %s record: %w", nr.Type, err)
	}

	// // NB(tlim): Commenting this out. In theory, .target is going away in v5.x and
	// this should be a no-op.
	//
	// if nr.Type == "NXDOMAIN" {
	// 	// The custom record's RDATA is empty, but legacy comparisons expect this target.
	// 	if err := rc.Set Target("NXDOMAIN"); err != nil {
	// 		return nil, fmt.Errorf("NXDOMAIN Set Target: %w", err)
	// 	}
	// }

	rc.Original = &nr

	// Read RouterOS-specific metadata fields applicable to ALL record types.
	if nr.MatchSubdomain == "true" || nr.MatchSubdomain == "yes" {
		rc.Metadata["match_subdomain"] = "true"
	}
	if nr.Regexp != "" {
		rc.Metadata["regexp"] = nr.Regexp
	}
	if nr.AddressList != "" {
		rc.Metadata["address_list"] = nr.AddressList
	}
	if nr.Comment != "" {
		rc.Metadata["comment"] = nr.Comment
	}

	return models.Records{rc}, nil
}

// recordToNative converts a dnscontrol RecordConfig to a RouterOS DNS static record for create/update.
func recordToNative(rc *models.RecordConfig) (*dnsStaticRecord, error) {
	nr := &dnsStaticRecord{
		Name: rc.GetLabelFQDN(),
		TTL:  formatMikrotikDuration(rc.TTL),
	}

	switch rc.Type {
	case "A":
		nr.Type = "A"
		nr.Address = rc.AsA().String()

	case "AAAA":
		nr.Type = "AAAA"
		nr.Address = rc.AsAAAA().String()

	case "CNAME":
		nr.Type = "CNAME"
		nr.CName = stripTrailingDot(rc.AsCNAME().Target)

	case "MIKROTIK_FWD":
		nr.Type = "FWD"
		nr.ForwardTo = rc.AsMIKROTIKFWD().ForwardTo

	case "MIKROTIK_NXDOMAIN":
		nr.Type = "NXDOMAIN"
		// NXDOMAIN has no target field — only name matters.

	case "MX":
		nr.Type = "MX"
		nr.MxExchange = stripTrailingDot(rc.AsMX().Mx)
		nr.MxPreference = strconv.FormatUint(uint64(rc.AsMX().Preference), 10)

	case "NS":
		nr.Type = "NS"
		nr.NS = stripTrailingDot(rc.AsNS().String())

	case "SRV":
		nr.Type = "SRV"
		srv := rc.AsSRV()
		nr.SrvTarget = stripTrailingDot(srv.Target)
		nr.SrvPort = strconv.FormatUint(uint64(srv.Port), 10)
		nr.SrvPriority = strconv.FormatUint(uint64(srv.Priority), 10)
		nr.SrvWeight = strconv.FormatUint(uint64(srv.Weight), 10)

	case "TXT":
		nr.Type = "TXT"
		nr.Text = rc.GetTargetTXTJoined()

	default:
		return nil, fmt.Errorf("mikrotik: unsupported record type %q", rc.Type)
	}

	// Write RouterOS-specific metadata fields applicable to ALL record types.
	// Always set these fields (even to empty) so the JSON payload explicitly
	// clears them on RouterOS when they are no longer desired.
	// match-subdomain is a boolean that RouterOS requires as "yes" or "no".
	if rc.Metadata["match_subdomain"] == "true" {
		nr.MatchSubdomain = "yes"
	} else {
		nr.MatchSubdomain = "no"
	}
	if rc.Metadata != nil {
		nr.Regexp = rc.Metadata["regexp"]
		nr.AddressList = rc.Metadata["address_list"]
		nr.Comment = rc.Metadata["comment"]
	}

	return nr, nil
}

func ensureTrailingDot(s string) string {
	if s == "" || strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

func stripTrailingDot(s string) string {
	return strings.TrimSuffix(s, ".")
}

// parseMikrotikDuration parses a RouterOS duration string like "1d", "10h", "15m", "30s",
// "1d00:00:00", "1w2d3h4m5s" into seconds.
func parseMikrotikDuration(s string) (uint32, error) {
	if s == "" {
		return 0, nil
	}

	// Try parsing as HH:MM:SS or NdHH:MM:SS format
	if m := reDurationHMS.FindStringSubmatch(s); m != nil {
		var total uint32
		if m[1] != "" {
			d, _ := strconv.ParseUint(m[1], 10, 32)
			total += uint32(d) * 86400
		}
		h, _ := strconv.ParseUint(m[2], 10, 32)
		minute, _ := strconv.ParseUint(m[3], 10, 32)
		sec, _ := strconv.ParseUint(m[4], 10, 32)
		total += uint32(h)*3600 + uint32(minute)*60 + uint32(sec)
		return total, nil
	}

	// Try parsing component format: 1w2d3h4m5s
	if m := reDurationComponents.FindStringSubmatch(s); m != nil {
		var total uint32
		if m[1] != "" {
			v, _ := strconv.ParseUint(m[1], 10, 32)
			total += uint32(v) * 604800
		}
		if m[2] != "" {
			v, _ := strconv.ParseUint(m[2], 10, 32)
			total += uint32(v) * 86400
		}
		if m[3] != "" {
			v, _ := strconv.ParseUint(m[3], 10, 32)
			total += uint32(v) * 3600
		}
		if m[4] != "" {
			v, _ := strconv.ParseUint(m[4], 10, 32)
			total += uint32(v) * 60
		}
		if m[5] != "" {
			v, _ := strconv.ParseUint(m[5], 10, 32)
			total += uint32(v)
		}
		return total, nil
	}

	return 0, fmt.Errorf("cannot parse RouterOS duration %q", s)
}

// formatMikrotikDuration converts seconds to a RouterOS-style duration string.
func formatMikrotikDuration(seconds uint32) string {
	if seconds == 0 {
		return "0s"
	}

	var parts []string
	if w := seconds / 604800; w > 0 {
		parts = append(parts, fmt.Sprintf("%dw", w))
		seconds %= 604800
	}
	if d := seconds / 86400; d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
		seconds %= 86400
	}
	if h := seconds / 3600; h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
		seconds %= 3600
	}
	if m := seconds / 60; m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
		seconds %= 60
	}
	if seconds > 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, "")
}

var (
	// Matches "1d00:00:00" or "00:00:00" format.
	reDurationHMS = regexp.MustCompile(`^(?:(\d+)d)?(\d{1,2}):(\d{2}):(\d{2})$`)
	// Matches "1w2d3h4m5s" component format (each part optional but at least one required).
	reDurationComponents = regexp.MustCompile(`^(?:(\d+)w)?(?:(\d+)d)?(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$`)
)

// ForwarderZone is the synthetic zone name used for managing RouterOS DNS forwarders.
const ForwarderZone = "_forwarders.mikrotik"

// forwarderToRecord converts a RouterOS DNS forwarder to a RecordConfig.
func forwarderToRecord(dc *models.DomainConfig, fwd dnsForwarder) *models.RecordConfig {
	rc := dc.MustNewRecordConfig(dc.LabelFromShort(fwd.Name), 300, privatetypes.TypeMIKROTIKFORWARDER, fwd.DNSServers)
	// Forwarders have no TTL; use dnscontrol's default to avoid spurious diffs.
	rc.Original = &fwd
	if fwd.DohServers != "" {
		rc.Metadata["doh_servers"] = fwd.DohServers
	}
	if fwd.VerifyDohCert == "true" {
		rc.Metadata["verify_doh_cert"] = "true"
	}
	return rc
}

// recordToForwarder converts a RecordConfig to a RouterOS DNS forwarder.
func recordToForwarder(rc *models.RecordConfig) *dnsForwarder {
	f := &dnsForwarder{
		Name:       rc.GetLabel(),
		DNSServers: rc.AsMIKROTIKFORWARDER().Target,
	}
	if rc.Metadata != nil {
		if v := rc.Metadata["doh_servers"]; v != "" {
			f.DohServers = v
		}
		if rc.Metadata["verify_doh_cert"] == "true" {
			f.VerifyDohCert = "true"
		}
	}
	return f
}

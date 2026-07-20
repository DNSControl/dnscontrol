package alidns

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"golang.org/x/net/idna"
)

// nativeToRecord converts an Alibaba Cloud DNS record to a RecordConfig.
func nativeToRecord(r *alidns.Record, domain string) (*models.RecordConfig, error) {
	rc := &models.RecordConfig{
		TTL:      uint32(r.TTL),
		Original: r,
	}

	label, err := idna.ToASCII(r.RR)
	if err != nil {
		return nil, fmt.Errorf("failed to convert label to ASCII: %w", err)
	}
	rc.SetLabel(label, domain)

	// Normalize CNAME, MX, NS records with trailing dot to be consistent with FQDN format.
	value := r.Value
	if r.Type == "CNAME" || r.Type == "MX" || r.Type == "NS" || r.Type == "SRV" {
		if value != "" && value != "." && !strings.HasSuffix(value, ".") {
			value = value + "."
		}
	}

	switch r.Type {
	case "MX":
		if err := rc.SetTargetMX(uint16(r.Priority), value); err != nil {
			return nil, fmt.Errorf("unparsable MX record received from ALIDNS: %w", err)
		}
	case "SRV":
		// SRV records in Alibaba Cloud: Value contains "priority weight port target"
		// e.g., "1 1 5060 www.cloud-example.com."
		// Parse the parts and normalize the target
		parts := strings.Fields(r.Value)
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid SRV format from ALIDNS: %s", r.Value)
		}
		target := parts[3]
		// Ensure target has trailing dot for FQDN
		if target != "" && target != "." && !strings.HasSuffix(target, ".") {
			target = target + "."
		}
		// Reconstruct with normalized target and let PopulateFromString handle it
		srvValue := fmt.Sprintf("%s %s %s %s", parts[0], parts[1], parts[2], target)
		if err := rc.PopulateFromString(r.Type, srvValue, domain); err != nil {
			return nil, fmt.Errorf("unparsable SRV record received from ALIDNS: %w", err)
		}
	case "CAA":
		// Alibaba Cloud CAA format: "0 issue \"letsencrypt.org\""
		if err := rc.SetTargetCAAString(r.Value); err != nil {
			return nil, fmt.Errorf("unparsable CAA record received from ALIDNS: %w", err)
		}
	case "TXT":
		if err := rc.SetTargetTXT(r.Value); err != nil {
			return nil, fmt.Errorf("unparsable TXT record received from ALIDNS: %w", err)
		}
	default:
		rc.Type = r.Type
		if err := rc.SetTarget(value); err != nil {
			return nil, fmt.Errorf("unparsable record received from ALIDNS: %w", err)
		}
	}

	return rc, nil
}

// recordToNativeContent converts a RecordConfig to the Value format expected by Alibaba Cloud DNS API.
func recordToNativeContent(r *models.RecordConfig) string {
	switch r.Type {
	case "SRV":
		return fmt.Sprintf("%d %d %d %s", r.SrvPriority, r.SrvWeight, r.SrvPort, r.GetTargetField())
	case "CAA":
		return fmt.Sprintf("%d %s \"%s\"", r.CaaFlag, r.CaaTag, r.GetTargetField())
	case "TXT":
		return r.GetTargetTXTJoined()
	default:
		return r.GetTargetField()
	}
}

// recordToNativePriority returns the priority value for MX and SRV records.
func recordToNativePriority(r *models.RecordConfig) int64 {
	switch r.Type {
	case "MX":
		return int64(r.MxPreference)
	case "SRV":
		return int64(r.SrvPriority)
	default:
		return 0
	}
}

// nativeToRecordNS takes a NS record from DNS and returns a native RecordConfig struct.
func nativeToRecordNS(ns string, origin string) *models.RecordConfig {
	rc := &models.RecordConfig{
		Type: "NS",
		TTL:  600,
	}
	rc.SetLabel("@", origin)
	rc.MustSetTarget(ns)
	return rc
}

package alidns

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"golang.org/x/net/idna"
)

// nativeToRecord converts an Alibaba Cloud DNS record to a RecordConfig.
func nativeToRecord(r *alidns.Record, dc *models.DomainConfig) (*models.RecordConfig, error) {
	label, err := idna.ToASCII(r.RR)
	if err != nil {
		return nil, fmt.Errorf("failed to convert label to ASCII: %w", err)
	}
	label = dc.LabelFromShort(label)

	// Normalize CNAME, MX, NS records with trailing dot to be consistent with FQDN format.
	value := r.Value
	if r.Type == "CNAME" || r.Type == "MX" || r.Type == "NS" || r.Type == "SRV" {
		if value != "" && value != "." && !strings.HasSuffix(value, ".") {
			value = value + "."
		}
	}

	ttl := uint32(r.TTL)
	var rc *models.RecordConfig
	switch r.Type {
	case "MX":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, r.Priority, value)
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
		// Reconstruct with normalized target and let NewRecordConfigParse handle it.
		srvValue := fmt.Sprintf("%s %s %s %s", parts[0], parts[1], parts[2], target)
		rc, err = dc.NewRecordConfigParse(label, ttl, dnsv2.TypeSRV, srvValue)
	case "CAA":
		// Alibaba Cloud CAA format: "0 issue \"letsencrypt.org\""
		rc, err = dc.NewRecordConfigParse(label, ttl, dnsv2.TypeCAA, r.Value)
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, r.Value)
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, value)
	}
	if err != nil {
		return nil, fmt.Errorf("unparsable %s record received from ALIDNS: %w", r.Type, err)
	}

	rc.Original = r
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
func nativeToRecordNS(ns string, dc *models.DomainConfig) (*models.RecordConfig, error) {
	return dc.NewRecordConfig("@", 600, dnsv2.TypeNS, ns)
}

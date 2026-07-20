package netbird

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
)

// nativeToRecordConfig converts a NetBird record to a dnscontrol RecordConfig.
func nativeToRecordConfig(domain string, r *Record) (*models.RecordConfig, error) {
	// NetBird API returns FQDNs, so we need to handle them properly
	name := r.Name

	// If the name doesn't end with a dot, it might be a FQDN from NetBird
	// Check if it already contains the domain
	if len(name) > 0 && name[len(name)-1] != '.' {
		// Name doesn't end with dot, check if it's already a FQDN
		if strings.HasSuffix(name, domain) {
			// FQDN, add the dot
			name = name + "."
		} else {
			// short name, use dnsutilv1.AddOrigin
			name = dnsutilv1.AddOrigin(r.Name, domain)
		}
	} else if len(name) > 0 && name[len(name)-1] == '.' {
		// FQDN, already has the dot, do nothing
	} else {
		// Empty name (apex record)
		name = dnsutilv1.AddOrigin(r.Name, domain)
	}

	target := r.Content
	// Make target FQDN for CNAME records
	if r.Type == "CNAME" {
		if target == "@" {
			target = domain
		}
		if target != "" && target[len(target)-1] != '.' {
			target = target + "."
		}
	}

	rc := &models.RecordConfig{
		Type:     r.Type,
		TTL:      uint32(r.TTL),
		Original: r,
	}
	rc.SetLabelFromFQDN(name, domain)

	switch r.Type {
	default:
		if err := rc.SetTarget(target); err != nil {
			return nil, err
		}
	}
	return rc, nil
}

// recordConfigToNative converts a dnscontrol RecordConfig to a NetBird record.
func recordConfigToNative(rc *models.RecordConfig, _ string) *CreateRecordRequest {
	// Remove trailing dot as NetBird API doesn't expect it
	name := rc.GetLabelFQDN()
	if len(name) > 0 && name[len(name)-1] == '.' {
		name = name[:len(name)-1]
	}

	target := rc.GetTargetField()

	switch rc.Type {
	case "CNAME":
		// Remove trailing dot
		if len(target) > 0 && target[len(target)-1] == '.' {
			target = target[:len(target)-1]
		}
	}

	return &CreateRecordRequest{
		Name:    name,
		Type:    rc.Type,
		Content: target,
		TTL:     int(rc.TTL),
	}
}

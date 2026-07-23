package netbird

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// nativeToRecordConfig converts a NetBird record to a dnscontrol RecordConfig.
func nativeToRecordConfig(dc *models.DomainConfig, r *Record) (*models.RecordConfig, error) {
	// NetBird API returns FQDNs, so we need to handle them properly
	name := r.Name
	var label string

	// If the name doesn't end with a dot, it might be a FQDN from NetBird
	// Check if it already contains the domain
	if len(name) > 0 && name[len(name)-1] != '.' {
		// Name doesn't end with dot, check if it's already a FQDN
		if strings.HasSuffix(name, dc.Name) {
			label = dc.LabelFromFQDNNoDot(name)
		} else {
			label = dc.LabelFromShort(name)
		}
	} else if len(name) > 0 && name[len(name)-1] == '.' {
		label = dc.LabelFromFQDNWithDot(name)
	} else {
		label = dc.LabelFromShort(name)
	}

	target := r.Content
	// Make target FQDN for CNAME records
	if r.Type == "CNAME" {
		if target == "@" {
			target = dc.Name
		}
		if target != "" && target[len(target)-1] != '.' {
			target = target + "."
		}
	}

	rc, err := dc.NewRecordConfig(label, uint32(r.TTL), r.Type, target)
	if err != nil {
		return nil, err
	}
	rc.Original = r
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

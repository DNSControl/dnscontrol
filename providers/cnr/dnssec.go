package cnr

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// isZoneSigned reports whether the DNS zone is DNSSEC-signed, based on the
// SIGNED property returned by StatusDNSZone.
func (n *Client) isZoneSigned(domain string) (bool, error) {
	r := n.client.Request(map[string]any{
		"COMMAND": "StatusDNSZone",
		"DNSZONE": domain,
	})
	if !r.IsSuccess() {
		return false, n.GetAPIError("Could not get status for DNS zone", domain, r)
	}
	signed, err := r.GetColumnIndex("SIGNED", 0)
	if err != nil {
		return false, fmt.Errorf("could not determine signed status for DNS zone %q: %w", domain, err)
	}
	return strings.TrimSpace(signed) == "1", nil
}

// getDNSSECCorrections returns the corrections needed to reconcile the zone's
// DNSSEC signing state with dc.AutoDNSSEC ("on", "off", or "" for no change).
func (n *Client) getDNSSECCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	if dc.AutoDNSSEC == "" {
		return nil, nil
	}
	signed, err := n.isZoneSigned(dc.Name)
	if err != nil {
		return nil, err
	}
	switch {
	case signed && dc.AutoDNSSEC == "off":
		return []*models.Correction{{
			Msg: fmt.Sprintf("Disable DNSSEC signing for zone %s", dc.Name),
			F:   n.setZoneSigned(dc.Name, false),
		}}, nil
	case !signed && dc.AutoDNSSEC == "on":
		return []*models.Correction{{
			Msg: fmt.Sprintf("Enable DNSSEC signing for zone %s", dc.Name),
			F:   n.setZoneSigned(dc.Name, true),
		}}, nil
	}
	return nil, nil
}

// setZoneSigned returns a function that enables or disables DNSSEC signing for
// the zone via ModifyDNSZone.
func (n *Client) setZoneSigned(domain string, signed bool) func() error {
	return func() error {
		value := "0"
		if signed {
			value = "1"
		}
		return n.updateZoneBy(map[string]any{"SIGNED": value}, domain)
	}
}

package ns1

import (
	"errors"
	"net/http"

	"github.com/DNSControl/dnscontrol/v4/models"
	"gopkg.in/ns1/ns1-go.v2/rest"
)

// GetZoneDNSSEC gets DNSSEC status for zone. Returns true for enabled, false for disabled
// a domain in NS1 can be in 3 states:
//  1. DNSSEC is enabled  (returns true)
//  2. DNSSEC is disabled (returns false)
//  3. some error state   (return false plus the error)
func (n *nsone) GetZoneDNSSEC(domain string) (bool, error) {
	for rtr := 0; ; rtr++ {
		_, httpResp, err := n.DNSSEC.Get(domain)
		// rest.ErrDNSECNotEnabled is our "disabled" state
		if err != nil && errors.Is(err, rest.ErrDNSECNotEnabled) {
			return false, nil
		}
		if httpResp.StatusCode == http.StatusTooManyRequests && rtr < clientRetries {
			continue
		}
		// any other errors not expected, let's surface them
		if err != nil {
			return false, err
		}

		// no errors returned, we assume DNSSEC is enabled
		return true, nil
	}
}

// getDomainCorrectionsDNSSEC creates DNSSEC zone corrections based on current state and preference.
func (n *nsone) getDomainCorrectionsDNSSEC(domain, toggleDNSSEC string) *models.Correction {
	// get dnssec status from NS1 for domain
	// if errors are returned, we bail out without any DNSSEC corrections
	status, err := n.GetZoneDNSSEC(domain)
	if err != nil {
		return nil
	}

	if toggleDNSSEC == "on" && !status {
		// disabled, but prefer it on, let's enable DNSSEC
		return &models.Correction{
			Msg: "ENABLE DNSSEC",
			F:   func() error { return n.configureDNSSEC(domain, true) },
		}
	} else if toggleDNSSEC == "off" && status {
		// enabled, but prefer it off, let's disable DNSSEC
		return &models.Correction{
			Msg: "DISABLE DNSSEC",
			F:   func() error { return n.configureDNSSEC(domain, false) },
		}
	}
	return nil
}

// configureDNSSEC configures DNSSEC for a zone. Set 'enabled' to true to enable, false to disable.
// There's a cornercase, in which DNSSEC is globally disabled for the account.
// In that situation, enabling DNSSEC will always fail with:
//
//	#1: ENABLE DNSSEC
//	FAILURE! POST https://api.nsone.net/v1/zones/example.com: 400 DNSSEC support is not enabled for this account. Please contact support@ns1.com to enable it
//
// Unfortunately this is not detectable otherwise, so given that we have a nice error message, we just let this through.
func (n *nsone) configureDNSSEC(domain string, enabled bool) error {
	z, _, err := n.Zones.Get(domain, true)
	if err != nil {
		return err
	}
	z.DNSSEC = &enabled
	for rtr := 0; ; rtr++ {
		httpResp, err := n.Zones.Update(z)
		if httpResp.StatusCode == http.StatusTooManyRequests && rtr < clientRetries {
			continue
		}
		return err
	}
}

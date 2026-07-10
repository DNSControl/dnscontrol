package ns1

import (
	"errors"
	"net/http"

	"github.com/DNSControl/dnscontrol/v4/models"
	"gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

func (n *nsone) ListZones() ([]string, error) {
	var zones []string

	for rtr := 0; ; rtr++ {
		zs, httpResp, err := n.Zones.List()
		if httpResp.StatusCode == http.StatusTooManyRequests && rtr < clientRetries {
			continue
		}
		if err != nil {
			return nil, err
		}

		for _, zone := range zs {
			zones = append(zones, zone.Zone)
		}
		return zones, nil
	}

}

// A wrapper around rest.Client's Zones.Get() implementing retries
// no explicit sleep is needed, it is implemented in NS1 client's RateLimitStrategy we used.
func (n *nsone) GetZone(domain string) (*dns.Zone, error) {
	for rtr := 0; ; rtr++ {
		z, httpResp, err := n.Zones.Get(domain, true)
		if httpResp.StatusCode == http.StatusTooManyRequests && rtr < clientRetries {
			continue
		}
		return z, err
	}
}

func (n *nsone) EnsureZoneExists(dc *models.DomainConfig) error {
	// This enables the create-domains subcommand
	zone := dns.NewZone(dc.Name)

	for rtr := 0; ; rtr++ {
		httpResp, err := n.Zones.Create(zone)
		if errors.Is(err, rest.ErrZoneExists) {
			// if domain exists already, just return nil, nothing to do here.
			return nil
		}
		// too many requests - retry w/out waiting. We specified rate limit strategy creating the client
		if httpResp.StatusCode == http.StatusTooManyRequests && rtr < clientRetries {
			continue
		}
		return err
	}
}

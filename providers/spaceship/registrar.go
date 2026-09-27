package spaceship

import (
	"fmt"
	"slices"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/namecheap/go-spaceship-sdk/client"
)

// GetRegistrarCorrections updates nameserver delegation at Spaceship.
func (c *spaceshipProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	info, err := c.getDomainInfo(dc.Name)
	if err != nil {
		return nil, err
	}

	existing := info.Nameservers.Hosts
	if len(existing) == 0 && strings.EqualFold(info.Nameservers.Provider, string(client.BasicNameserverProvider)) {
		existing = defaultNS
	}

	desired := models.NameserversToStrings(dc.Nameservers)
	existingNorm := normalizeHosts(existing)
	desiredNorm := normalizeHosts(desired)
	if slices.Equal(existingNorm, desiredNorm) {
		return nil, nil
	}

	req := nameserverUpdate(desiredNorm)
	return []*models.Correction{
		{
			Msg: fmt.Sprintf("Update nameservers [%s] -> [%s]", strings.Join(existingNorm, ","), strings.Join(desiredNorm, ",")),
			F: func() error {
				return c.updateDomainNameServers(dc.Name, req)
			},
		},
	}, nil
}

func nameserverUpdate(desired []string) client.UpdateNameserverRequest {
	if slices.Equal(desired, normalizeHosts(defaultNS)) {
		return client.UpdateNameserverRequest{Provider: client.BasicNameserverProvider}
	}
	return client.UpdateNameserverRequest{
		Provider: client.CustomNameserverProvider,
		Hosts:    desired,
	}
}

func normalizeHosts(hosts []string) []string {
	out := make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
		if host != "" {
			out = append(out, host)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

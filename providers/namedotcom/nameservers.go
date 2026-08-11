package namedotcom

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/namedotcom/go/namecom"
)

var nsRegex = regexp.MustCompile(`ns([1-4])[a-z]{3}\.name\.com`)

// GetNameservers gets the nameservers set on a domain.
func (n *namedotcomProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	// This is an interesting edge case. Name.com expects you to SET the nameservers to ns[1-4].name.com,
	// but it will internally set it to ns1xyz.name.com, where xyz is a uniqueish 3 letters.
	// In order to avoid endless loops, we will use the unique nameservers if present, or else the generic ones if not.
	nss, err := n.getNameserversRaw(domain)
	if err != nil {
		return nil, err
	}
	toUse := []string{"ns1.name.com", "ns2.name.com", "ns3.name.com", "ns4.name.com"}
	for _, ns := range nss {
		if matches := nsRegex.FindStringSubmatch(ns); len(matches) == 2 && len(matches[1]) == 1 {
			idx := matches[1][0] - '1' // regex ensures proper range
			toUse[idx] = matches[0]
		}
	}
	return models.ToNameservers(toUse)
}

func (n *namedotcomProvider) getNameserversRaw(domain string) ([]string, error) {
	request := &namecom.GetDomainRequest{
		DomainName: domain,
	}

	response, err := n.client.GetDomain(request)
	if err != nil {
		return nil, err
	}

	sort.Strings(response.Nameservers)
	return response.Nameservers, nil
}

// GetRegistrarCorrections gathers corrections that would bring n to match dc.
func (n *namedotcomProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	nss, err := n.getNameserversRaw(dc.Name)
	if err != nil {
		return nil, err
	}
	foundNameservers := strings.Join(nss, ",")
	expected := []string{}
	for _, ns := range dc.Nameservers {
		expected = append(expected, ns.Name)
	}
	sort.Strings(expected)
	expectedNameservers := strings.Join(expected, ",")

	if foundNameservers != expectedNameservers {
		return []*models.Correction{
			{
				Msg: fmt.Sprintf("Update nameservers %s -> %s", foundNameservers, expectedNameservers),
				F:   n.updateNameservers(expected, dc.Name),
			},
		}, nil
	}
	return nil, nil
}

func (n *namedotcomProvider) updateNameservers(ns []string, domain string) func() error {
	return func() error {
		request := &namecom.SetNameserversRequest{
			DomainName:  domain,
			Nameservers: ns,
		}

		_, err := n.client.SetNameservers(request)
		return err
	}
}

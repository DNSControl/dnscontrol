package ns1

import (
	"errors"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"gopkg.in/ns1/ns1-go.v2/rest"
)

func (n *nsone) GetNameservers(domain string) ([]*models.Nameserver, error) {
	var nservers []string

	z, err := n.GetZone(domain)
	if err != nil && errors.Is(err, rest.ErrZoneMissing) {
		// if we get here, zone wasn't created, but we ended up continuing regardless.
		// This should be revisited, but for now let's get out early with a relevant message
		// one case: preview --no-populate
		printer.Warnf("GetNameservers: Zone %s not created in NS1. Either create manually or ensure dnscontrol can create it.\n", domain)
		return models.ToNameservers(nservers)
	}

	if err != nil {
		return nil, err
	}

	// on newly-created domains NS1 may assign nameservers with or without a
	// trailing dot. This is not reflected in the actual DNS records, that
	// always have the trailing dots.
	//
	// Handle both scenarios by stripping dots where existing, before continuing.
	for _, ns := range z.DNSServers {
		if strings.HasSuffix(ns, ".") {
			nservers = append(nservers, ns[0:len(ns)-1])
		} else {
			nservers = append(nservers, ns)
		}
	}
	return models.ToNameservers(nservers)
}

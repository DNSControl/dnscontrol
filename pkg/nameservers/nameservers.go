// Package nameservers provides logic for dynamically finding nameservers for a domain, and configuring NS records for them.
package nameservers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

// DetermineNameservers will find all nameservers we should use for a domain. It follows the following rules:
// 1. All explicitly defined NAMESERVER records will be used.
// 2. Each DSP declares how many nameservers to use. Default is all. 0 indicates to use none.
func DetermineNameservers(dc *models.DomainConfig) ([]*models.Nameserver, error) {
	return DetermineNameserversForProviders(dc, dc.DNSProviderInstances, false)
}

// DetermineNameserversForProviders is like DetermineNameservers, for a subset of providers.
func DetermineNameserversForProviders(dc *models.DomainConfig, providers []*models.DNSProviderInstance, silent bool) ([]*models.Nameserver, error) {
	// start with the nameservers that have been explicitly added:
	ns := dc.Nameservers

	for _, dnsProvider := range providers {
		n := dnsProvider.NumberOfNameservers
		if n == 0 {
			continue
		}
		if !silent && !printer.SkinnyReport {
			fmt.Printf("----- Getting nameservers from: %s\n", dnsProvider.Name)
		}

		nss, err := dnsProvider.Driver.GetNameservers(dc.Name)
		if err != nil {
			return nil, fmt.Errorf("error while getting Nameservers for zone=%q with provider=%q: %w", dc.Name, dnsProvider.Name, err)
		}
		// Clean up the nameservers due to
		// https://github.com/DNSControl/dnscontrol/issues/491
		// In the far future, this warning will become a fatal error.
		for i := range nss {
			if strings.HasSuffix(nss[i].Name, ".") {
				models.WarnNameserverDot(dnsProvider.Name, fmt.Sprintf("DetermineNameservers (%s) (%s)", dc.Name, nss[i].Name))
				nss[i].Name = strings.TrimSuffix(nss[i].Name, ".")
			}
		}

		take := len(nss)
		if n > 0 && n < take {
			take = n
		}
		for i := range take {
			ns = append(ns, nss[i])
		}
	}
	return ns, nil
}

// AddNSRecords creates NS records on a domain corresponding to the nameservers specified.
func AddNSRecords(dc *models.DomainConfig) {
	ttl := uint32(300)
	if ttls, ok := dc.Metadata["ns_ttl"]; ok {
		t, err := strconv.ParseUint(ttls, 10, 32)
		if err != nil {
			fmt.Printf("WARNING: ns_ttl for %s (%s) is not a valid int", dc.Name, ttls)
		} else {
			ttl = uint32(t)
		}
	}
	for _, ns := range dc.Nameservers {
		rc := &models.RecordConfig{
			Type:     "NS",
			Metadata: map[string]string{},
			TTL:      ttl,
		}
		rc.SetLabel("@", dc.Name)
		t := ns.Name
		if !strings.HasSuffix(t, ".") {
			t += "."
		}
		if err := rc.SetTarget(t); err != nil {
			fmt.Printf("failed AddNSRecords rc.SetTarget(%q): %s\n", t, err)
		}

		dc.Records = append(dc.Records, rc)
	}
}

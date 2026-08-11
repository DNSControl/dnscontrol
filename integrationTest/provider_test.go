package main

// Test the providers.

import (
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/nameservers"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/DNSControl/dnscontrol/v4/pkg/zonerecs"
)

// TestDualProviders verifies that providers labeled DocDualHost support having
// delegation to multiple DNS Providers. i.e. verifies that example.com's
// delegations can be both ROUTE53 and GCLOUD. Some providers want to be the
// only provider for a domain.
func TestDualProviders(t *testing.T) {
	p, domain, _ := getProvider(t)
	if p == nil {
		return
	}
	if domain == "" {
		t.Fatal("NO DOMAIN SET!  Exiting!")
	}
	dc := getDomainConfigWithNameservers(t, p, domain)
	if !providers.ProviderHasCapability(*providerFlag, providers.DocDualHost) {
		t.Skip("Skipping.  DocDualHost == Cannot")
		return
	}
	// clear everything
	run := func() {
		dom, _ := dc.Copy()

		rs, cs, _, err := zonerecs.CorrectZoneRecords(p, dom)
		if err != nil {
			t.Fatal(err)
		}
		for i, c := range rs {
			t.Logf("INFO#%d:\n%s", i+1, c.Msg)
		}
		for i, c := range cs {
			t.Logf("#%d:\n%s", i+1, c.Msg)
			if err = c.F(); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Log("Clearing everything")
	run()
	// add bogus nameservers
	dc.Records = []*models.RecordConfig{}
	nsList, _ := models.ToNameservers([]string{"ns1.example.com", "ns2.example.com"})
	dc.Nameservers = append(dc.Nameservers, nsList...)
	nameservers.AddNSRecords(dc)
	t.Log("Adding test nameservers")
	run()
	// run again to make sure no corrections
	t.Log("Running again to ensure stability")
	rs, cs, actualChangeCount, err := zonerecs.CorrectZoneRecords(p, dc)
	if err != nil {
		t.Fatal(err)
	}
	if actualChangeCount != 0 {
		t.Logf("Expect no corrections on second run, but found %d.", actualChangeCount)
		for i, c := range rs {
			t.Logf("INFO#%d:\n%s", i+1, c.Msg)
		}
		for i, c := range cs {
			t.Logf("#%d: %s", i+1, c.Msg)
		}
		t.FailNow()
	}

	t.Log("Removing test nameservers")
	dc.Records = []*models.RecordConfig{}
	n := 0
	for _, ns := range dc.Nameservers {
		if ns.Name == "ns1.example.com" || ns.Name == "ns2.example.com" {
			continue
		}
		dc.Nameservers[n] = ns
		n++
	}
	dc.Nameservers = dc.Nameservers[:n]
	nameservers.AddNSRecords(dc)
	run()
}

// TestNameserverDots verifies a provider returns properly-formed nameservers.
func TestNameserverDots(t *testing.T) {
	// Issue https://github.com/DNSControl/dnscontrol/issues/491
	// If this fails, the provider's GetNameservers() function uses
	// models.ToNameserversStripTD() instead of models.ToNameservers()
	// or vise-versa.

	// Setup:
	p, domain, _ := getProvider(t)
	if p == nil {
		return
	}
	if domain == "" {
		t.Fatal("NO DOMAIN SET!  Exiting!")
	}
	dc := getDomainConfigWithNameservers(t, p, domain)
	if !providers.ProviderHasCapability(*providerFlag, providers.DocDualHost) {
		t.Skip("Skipping.  DocDualHost == Cannot")
		return
	}

	t.Run("No trailing dot in nameserver", func(t *testing.T) {
		for _, nameserver := range dc.Nameservers {
			// fmt.Printf("DEBUG: nameserver.Name = %q\n", nameserver.Name)
			if strings.HasSuffix(nameserver.Name, ".") {
				t.Errorf("Provider returned nameserver with trailing dot: %q", nameserver)
			}
		}
	})
}

// TestDuplicateNameservers verifies that a provider de-dupes nameservers if existing nameservers are added.
func TestDuplicateNameservers(t *testing.T) {
	// Issue: https://github.com/DNSControl/dnscontrol/issues/3088
	// Only configuring for Azure DNS

	// Setup:
	p, domain, cfg := getProvider(t)
	if p == nil {
		return
	}

	if domain == "" {
		t.Fatal("NO DOMAIN SET!  Exiting!")
	}

	dc := getDomainConfigWithNameservers(t, p, domain)

	if !providers.ProviderHasCapability(*providerFlag, providers.DocDualHost) {
		t.Skip("Skipping.  DocDualHost == Cannot")
		return
	}

	if cfg["TYPE"] != "AZURE_DNS" {
		t.Skip("Skipping. Deduplication logic is not implemented for this provider.")
		return
	}

	// clear everything
	run := func(expectedChangeCount int, msg string, ignoreExpected bool) {
		dom, _ := dc.Copy()

		rs, cs, actualChangeCount, err := zonerecs.CorrectZoneRecords(p, dom)
		if err != nil {
			t.Fatal(err)
		}
		for i, c := range rs {
			t.Logf("INFO#%d:\n%s", i+1, c.Msg)
		}
		for i, c := range cs {
			t.Logf("#%d:\n%s", i+1, c.Msg)
			if err = c.F(); err != nil {
				t.Fatal(err)
			}
		}
		if (!ignoreExpected) && actualChangeCount != expectedChangeCount {
			t.Logf(msg, actualChangeCount)
			t.FailNow()
		}
	}

	t.Log("Clearing everything")
	dc.Records = []*models.RecordConfig{}
	n := 0
	for _, ns := range dc.Nameservers {
		if ns.Name == "ns1.example.com" || ns.Name == "ns2.example.com" {
			continue
		}
		dc.Nameservers[n] = ns
		n++
	}
	dc.Nameservers = dc.Nameservers[:n]
	nameservers.AddNSRecords(dc)
	run(0, "Clearing everything", true)

	// add bogus nameservers and duplicate nameservers
	dc.Records = []*models.RecordConfig{}
	nsList, _ := models.ToNameservers([]string{"ns1.example.com"})
	dc.Nameservers = append(dc.Nameservers, dc.Nameservers...)
	dc.Nameservers = append(dc.Nameservers, nsList...)
	nameservers.AddNSRecords(dc)
	t.Log("Adding test nameservers")
	run(1, "Expect 1 correction, but found %d.", false)

	// run again to make sure no corrections
	t.Log("Running again to ensure stability")
	run(0, "Expect no corrections on second run, but found %d.", false)

	t.Log("Removing test nameservers")
	dc.Records = []*models.RecordConfig{}
	n = 0
	for _, ns := range dc.Nameservers {
		if ns.Name == "ns1.example.com" || ns.Name == "ns2.example.com" {
			continue
		}
		dc.Nameservers[n] = ns
		n++
	}
	dc.Nameservers = dc.Nameservers[:n]
	nameservers.AddNSRecords(dc)
	run(0, "Removing test nameservers", true)
}

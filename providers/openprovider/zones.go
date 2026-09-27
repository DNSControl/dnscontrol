package openprovider

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"golang.org/x/net/publicsuffix"
)

// EnsureZoneExists creates a standard master DNS zone when it is absent.
func (p *openproviderProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	_, err := p.client.getZone(dc.Name)
	if err == nil {
		return nil
	}
	if !isNotFound(err) {
		return err
	}

	domain, err := splitZoneName(dc.Name)
	if err != nil {
		return err
	}
	return p.client.createZone(createZoneRequest{
		Domain:               domain,
		Type:                 "master",
		IsSpamExpertsEnabled: "off",
		Secured:              false,
	})
}

func splitZoneName(name string) (zoneDomain, error) {
	name = strings.TrimSuffix(strings.ToLower(name), ".")
	extension, _ := publicsuffix.PublicSuffix(name)
	base := strings.TrimSuffix(name, "."+extension)
	if extension == "" || base == "" || base == name {
		return zoneDomain{}, fmt.Errorf("OPENPROVIDER: cannot split DNS zone name %q into name and extension", name)
	}
	return zoneDomain{Name: base, Extension: extension}, nil
}

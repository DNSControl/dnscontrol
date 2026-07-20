package joker

import "github.com/DNSControl/dnscontrol/v4/models"

// GetNameservers returns the nameservers for a domain.
func (api *jokerProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	// For DNS-only providers like Joker, we can return an empty list
	// since nameserver management is typically handled separately
	return []*models.Nameserver{}, nil
}

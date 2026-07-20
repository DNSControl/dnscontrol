package rejectif

import (
	"errors"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// Keep these in alphabetical order.

// NsAtApex detects NS records at the apex/root domain.
// Use this when a provider doesn't support custom NS records at the apex.
func NsAtApex(rc *models.RecordConfig) error {
	if rc.GetLabel() == "" {
		return errors.New("NS records not supported at apex")
	}
	return nil
}

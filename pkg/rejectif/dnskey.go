package rejectif

import (
	"errors"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// Keep these in alphabetical order.

// DnskeyNotAtApex detects DNSKEY records not at the apex/root domain.
// Use this when a provider doesn't support custom DNSKEY records at subnames.
func DnskeyNotAtApex(rc *models.RecordConfig) error {
	if rc.GetLabel() != "" && rc.GetLabel() != "@" {
		return errors.New("DNSKEY records not supported at subnames")
	}
	return nil
}

package rejectif

import (
	"errors"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// Keep these in alphabetical order.

// NaptrHasEmptyTarget detects NAPTR records with empty targets.
func NaptrHasEmptyTarget(rc *models.RecordConfig) error {
	if rc.AsNAPTR().Replacement == "" {
		return errors.New("naptr has empty target")
	}
	return nil
}

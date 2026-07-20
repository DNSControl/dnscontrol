package rejectif

import (
	"errors"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// Keep these in alphabetical order.

// NaptrHasEmptyTarget detects NAPTR records with empty targets.
func NaptrHasEmptyTarget(rc *models.RecordConfig) error {
	if rc.GetTargetField() == "" {
		return errors.New("naptr has empty target")
	}
	return nil
}

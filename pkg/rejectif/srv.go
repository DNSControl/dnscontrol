package rejectif

import (
	"errors"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// Keep these in alphabetical order.

// SrvHasNullTarget detects SRV records that has a null target.
func SrvHasNullTarget(rc *models.RecordConfig) error {
	if rc.GetTargetField() == "." {
		return errors.New("srv has empty target") // Same msg as SrvHasEmptyTarget to make testing easier.
	}
	return nil
}

// SrvHasEmptyTarget detects SRV records with empty targets.
func SrvHasEmptyTarget(rc *models.RecordConfig) error {
	if rc.GetTargetField() == "" {
		return errors.New("srv has empty target")
	}
	return nil
}

// SrvHasZeroPort detects SRV records with port set to zero.
func SrvHasZeroPort(rc *models.RecordConfig) error {
	if rc.SrvPort == 0 {
		return errors.New("srv has zero port")
	}
	return nil
}

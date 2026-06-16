package rejectif

import (
	"errors"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// Keep these in alphabetical order.

// LabelIsWildcard detects wildcard labels (e.g. "*.example.com").
func LabelIsWildcard(rc *models.RecordConfig) error {
	if strings.HasPrefix(rc.GetLabel(), "*") {
		return errors.New("wildcard labels are not supported")
	}
	return nil
}

// LabelNotApex detects use not at apex. Use this when a record type
// is only permitted at the apex.
func LabelNotApex(rc *models.RecordConfig) error {
	if rc.GetLabel() != "@" {
		return errors.New("use not at apex")
	}
	return nil
}

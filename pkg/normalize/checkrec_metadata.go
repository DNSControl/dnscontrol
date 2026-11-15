package normalize

import (
	"errors"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// checkObsoleteMetadata rejects metadata that is no longer supported.
func checkObsoleteMetadata(rec *models.RecordConfig) error {
	if _, ok := rec.Metadata["ignore_name_disable_safety_check"]; ok {
		return errors.New("IGNORE_NAME_DISABLE_SAFETY_CHECK no longer supported. Please use DISABLE_IGNORE_SAFETY_CHECK for the entire domain")
	}
	return nil
}

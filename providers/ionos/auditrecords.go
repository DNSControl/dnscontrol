package ionos

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// minTTL is the lowest TTL the API accepts; below it answers HTTP 400.
const minTTL = 60

// AuditRecords returns a list of errors corresponding to the records
// that aren't supported by this provider. If all records are
// supported, an empty list is returned.
func (*ionosProvider) AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	// Targets are sent without their trailing dot, so the root has no representation.
	a.Add("MX", rejectif.MxNull)
	a.Add("SRV", rejectif.SrvHasNullTarget)
	a.Add("CNAME", targetIsRoot)
	a.Add("NS", targetIsRoot)

	// The API returns TXT values quoted and its documentation does not describe how
	// it escapes them, so these values could not be compared reliably.
	a.Add("TXT", rejectif.TxtHasDoubleQuotes)
	a.Add("TXT", rejectif.TxtHasBackslash)
	a.Add("TXT", rejectif.TxtIsEmpty)

	a.Add("*", ttlTooLow)

	return a.Audit(records)
}

func ttlTooLow(rc *models.RecordConfig) error {
	if rc.TTL < minTTL {
		return fmt.Errorf("IONOS requires a TTL of at least %d, got %d", minTTL, rc.TTL)
	}
	return nil
}

func targetIsRoot(rc *models.RecordConfig) error {
	if rc.GetRDATA().String() == "." {
		return fmt.Errorf("IONOS cannot store the root as the target of a %s record", rc.Type)
	}
	return nil
}

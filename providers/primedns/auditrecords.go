package primedns

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

// AuditRecords returns a list of errors corresponding to the records that PrimeDNS cannot store.
// If all records are supported, an empty list is returned.
func (*primednsProvider) AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}

	a.Add("NS", apexNSTTL)

	errs := a.Audit(records)
	for _, rc := range records {
		if rc.TTL < 60 || rc.TTL > 86400 {
			errs = append(errs, fmt.Errorf("%s %s: PrimeDNS TTL must be between 60 and 86400 seconds, got %d", rc.GetLabelFQDN(), rc.Type, rc.TTL))
		}
	}
	return errs
}

// systemNSTTL is the TTL PrimeDNS gives the apex NS records it creates with a zone.
const systemNSTTL = 8000

// apexNSTTL checks the TTL of an NS record at the zone apex. PrimeDNS keeps all records of a name and type on one TTL
// (RFC 2181 section 5.2) and the provider replaces the whole apex NS RRset, PrimeDNS's own records included. A different
// TTL would change the TTL of PrimeDNS's own NS records too (measured against the API on 2026-11-09: a PUT with TTL 300
// was accepted and moved all apex NS records to 300), so an extra apex NS record (a second provider) has to use the
// same TTL as the records PrimeDNS created.
func apexNSTTL(rc *models.RecordConfig) error {
	if rc.GetLabel() == "@" && rc.TTL != systemNSTTL {
		return fmt.Errorf("PrimeDNS needs the same TTL for all NS records at the apex, %d seconds like its own; add TTL(%d) to this NS record", systemNSTTL, systemNSTTL)
	}
	return nil
}

package openprovider

import (
	"fmt"
	"slices"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/rejectif"
)

var supportedRecordTypes = []string{
	"A",
	"AAAA",
	"CAA",
	"CNAME",
	"MX",
	"NS",
	"SRV",
	"TLSA",
	"TXT",
}

var supportedCAATags = []string{"issue", "issuewild", "iodef"}

// AuditRecords returns errors for records that Openprovider cannot represent.
func AuditRecords(records models.Records) []error {
	a := rejectif.Auditor{}
	a.TypesSupported(supportedRecordTypes)
	a.Add("NS", rejectUnsupportedNS)          // Last verified 2026-09-21
	a.Add("MX", rejectif.MxNull)              // Last verified 2026-09-21
	a.Add("SRV", rejectif.SrvHasNullTarget)   // Last verified 2026-09-21
	a.Add("TXT", rejectif.TxtIsEmpty)         // Last verified 2026-09-21
	a.Add("TXT", rejectif.TxtHasDoubleQuotes) // Last verified 2026-09-21
	a.Add("TXT", rejectif.TxtHasBackslash)    // Last verified 2026-09-21
	a.Add("CAA", rejectUnsupportedCAAFlag)    // Last verified 2026-09-21
	a.Add("CAA", rejectUnsupportedCAATag)     // Last verified 2026-09-21
	a.Add("CAA", rejectUnsupportedCAAFields)  // Last verified 2026-09-21
	return a.Audit(records)
}

func rejectUnsupportedNS(rc *models.RecordConfig) error {
	if rc.GetLabel() != apexLabel {
		return fmt.Errorf("NS records are only supported at the Openprovider zone apex")
	}
	return nil
}

func rejectUnsupportedCAAFlag(rc *models.RecordConfig) error {
	if rc.AsCAA().Flag != 0 {
		return fmt.Errorf("CAA flag %d is not supported by Openprovider", rc.AsCAA().Flag)
	}
	return nil
}

func rejectUnsupportedCAATag(rc *models.RecordConfig) error {
	if !slices.Contains(supportedCAATags, rc.AsCAA().Tag) {
		return fmt.Errorf("CAA tag %q is not supported by Openprovider", rc.AsCAA().Tag)
	}
	return nil
}

func rejectUnsupportedCAAFields(rc *models.RecordConfig) error {
	if strings.Contains(rc.AsCAA().Value, ";") {
		return fmt.Errorf("CAA target fields are not supported by Openprovider")
	}
	return nil
}

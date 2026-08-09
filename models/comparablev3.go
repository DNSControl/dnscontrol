package models

import (
	"fmt"
	"strings"
)

// generateComparableV3 generates or regenerates the .ComparableV3 field from the current .RDATA. It does not modify .RDATA.
func (rc *RecordConfig) generateComparableV3() {
	switch rc.Type {

	case "IGNORE":
		return

	case "IMPORT_TRANSFORM":
		return

	case "SOA":
		// The comparable string for SOA intentionally excludes the serial
		// number, because the serial number changes on every update and
		// would prevent correct diffing. List it as "X" so-as it stands out
		// in debug output that the serial is intentionally excluded.
		rd := rc.AsSOA()
		rc.ComparableV3 = fmt.Sprintf("%s %s X %d %d %d %d", rd.Ns, rd.Mbox, rd.Refresh, rd.Retry, rd.Expire, rd.Minttl)

	default:
		if rc.GetRDATA() == nil {
			panic(fmt.Sprintf("BUG: FixUp: .RDATA is nil for type %s", rc.Type))
		}
		rc.ComparableV3 = strings.TrimSpace(rc.GetRDATA().String())
	}
}

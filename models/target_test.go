package models

import "testing"

func TestGetTargetFieldEmptyRDATA(t *testing.T) {
	dc := MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("blocked", 300, "MIKROTIK_NXDOMAIN")
	if got := rc.GetTargetField(); got != "" {
		t.Fatalf("GetTargetField() = %q, want empty string", got)
	}
}

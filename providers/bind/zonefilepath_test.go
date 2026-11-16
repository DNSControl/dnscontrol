package bind

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func Test_zoneFilePath(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		domain  string
		want    string
		wantErr bool
	}{
		{"simple", "%c.zone", "example.com", "zones/example.com.zone", false},
		{"rfc2317", "%c.zone", "0/26.2.0.192.in-addr.arpa", "zones/0/26.2.0.192.in-addr.arpa.zone", false},
		{"dotdot", "%c.zone", "../../outside/deep/owned", "", true},
		{"dotdot only", "%c", "..", "", true},
		{"dotdot inner", "%c.zone", "a/../../owned", "", true},
		{"absolute", "%c.zone", "/etc/owned", "", true},
		{"raw verb", "%r.zone", "../owned", "", true},
		{"legacy verb", "%D.zone", "../owned", "", true},
		{"prefix keeps it local", "db_%D", "../owned", "zones/db_../owned", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &bindProvider{directory: "zones", filenameformat: tt.format}
			dc := &models.DomainConfig{
				Name:        tt.domain,
				NameRaw:     tt.domain,
				NameUnicode: tt.domain,
				UniqueName:  tt.domain,
			}
			got, err := c.zoneFilePath(dc)
			if (err != nil) != tt.wantErr {
				t.Fatalf("zoneFilePath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != filepath.FromSlash(tt.want) {
				t.Errorf("zoneFilePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test_traversalWritesNothing verifies that a zone name containing ".."
// is rejected before anything is read or written. (GHSA-rrxr-r9rr-crr4)
func Test_traversalWritesNothing(t *testing.T) {
	top := t.TempDir()
	c := &bindProvider{directory: filepath.Join(top, "zones"), filenameformat: "%c.zone"}
	name := "../outside/deep/owned"
	dc := &models.DomainConfig{Name: name, NameRaw: name, NameUnicode: name, UniqueName: name}

	if _, err := c.GetZoneRecords(dc); err == nil {
		t.Error("GetZoneRecords() expected error, got nil")
	}
	if _, _, err := c.GetZoneRecordsCorrections(dc, nil); err == nil {
		t.Error("GetZoneRecordsCorrections() expected error, got nil")
	}
	if _, err := os.Stat(filepath.Join(top, "outside")); !os.IsNotExist(err) {
		t.Errorf("directory outside of the zones dir was created (err=%v)", err)
	}
}

package commands

import (
	"errors"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func Test_whichZonesToProcess(t *testing.T) {

	dcNoTag := models.MustNewDomainConfig("example.com")
	dcNoTag2 := models.MustNewDomainConfig("example.net")
	dcTaggedEmpty := models.MustNewDomainConfig("example.com!")
	dcTaggedGeorge := models.MustNewDomainConfig("example.com!george")
	dcTaggedJohn := models.MustNewDomainConfig("example.com!john")

	allDC := []*models.DomainConfig{
		dcNoTag,
		dcNoTag2,
		dcTaggedGeorge,
		dcTaggedJohn,
		dcTaggedEmpty,
	}

	// This is needed since we aren't calling js.ExecuteJavaScript().
	for _, dc := range allDC {
		dc.PostProcess()
	}

	type args struct {
		dc     []*models.DomainConfig
		filter string
	}

	tests := []struct {
		name string
		why  string
		args args
		want []*models.DomainConfig
	}{
		{
			name: "testAllFilter",
			why:  "Should return all domain configs",
			args: args{
				dc:     allDC,
				filter: "all",
			},
			want: allDC,
		},
		{
			name: "testNoFilter",
			why:  "Should return all domain configs",
			args: args{
				dc:     allDC,
				filter: "",
			},
			want: allDC,
		},
		{
			name: "testFilterTagged",
			why:  "Should return one tagged domain",
			args: args{
				dc:     allDC,
				filter: "example.com!george",
			},
			want: []*models.DomainConfig{dcTaggedGeorge},
		},
		{
			name: "testMultiFilterTagged",
			why:  "Should return two tagged domains",
			args: args{
				dc:     allDC,
				filter: "example.com!george,example.com!john",
			},
			want: []*models.DomainConfig{dcTaggedGeorge, dcTaggedJohn},
		},
		{
			name: "testMultiFilterTaggedNoMatch",
			why:  "Should return nothing",
			args: args{
				dc:     allDC,
				filter: "example.com!ringo",
			},
			want: []*models.DomainConfig{},
		},
		{
			name: "testMultiFilterTaggedWildcard",
			why:  "Should return all matching tagged domains",
			args: args{
				dc:     allDC,
				filter: "example.com!*",
			},
			want: []*models.DomainConfig{dcTaggedGeorge, dcTaggedJohn},
		},
		{
			name: "testFilterNoTag",
			why:  "Should return untagged and empty tagged domain",
			args: args{
				dc:     allDC,
				filter: "example.com",
			},
			want: []*models.DomainConfig{dcNoTag, dcTaggedEmpty},
		},
		{
			name: "testFilterEmptyTag",
			why:  "Should return untagged and empty tagged domain",
			args: args{
				dc:     allDC,
				filter: "example.com!",
			},
			want: []*models.DomainConfig{dcNoTag, dcTaggedEmpty},
		},
		{
			name: "testFilterEmptyTagAndNoTag",
			why:  "Should return untagged and empty tagged domain",
			args: args{
				dc:     allDC,
				filter: "example.com!,example.com",
			},
			want: []*models.DomainConfig{dcNoTag, dcTaggedEmpty},
		},
		{
			name: "testFilterNoTagTagged",
			why:  "Should return the tagged and untagged domains",
			args: args{
				dc:     allDC,
				filter: "example.com!george,example.com",
			},
			want: []*models.DomainConfig{dcTaggedGeorge, dcNoTag, dcTaggedEmpty},
		},
		{
			name: "testFilterDuplicates2",
			why:  "Should return one untagged domain",
			args: args{
				dc:     allDC,
				filter: "example.net,example.net",
			},
			want: []*models.DomainConfig{dcNoTag2},
		},
		{
			name: "testFilterNoTagNoMatch",
			why:  "Should return nothing",
			args: args{
				dc:     []*models.DomainConfig{dcTaggedGeorge, dcTaggedJohn},
				filter: "example.com",
			},
			want: []*models.DomainConfig{},
		},
		{
			name: "testFilterTaggedNoMatch",
			why:  "Should return nothing",
			args: args{
				dc:     []*models.DomainConfig{dcNoTag},
				filter: "example.com!george",
			},
			want: []*models.DomainConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := whichZonesToProcess(tt.args.dc, tt.args.filter)
			if len(got) != len(tt.want) {
				t.Errorf("whichZonesToProcess() %s: %s", tt.name, tt.why)
				for i := range got {
					t.Errorf("got[%d]: %s", i, got[i].GetUniqueName())
				}
				for i := range tt.want {
					t.Errorf("want[%d]: %s", i, tt.want[i].GetUniqueName())
				}
				return
			}
			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("whichZonesToProcess() %s: %s", tt.name, tt.why)
					return
				}
			}
		})
	}
}

func Test_zoneWillBeCreated(t *testing.T) {
	const provider = "cf"

	tests := []struct {
		name        string
		corrections []*models.Correction
		want        bool
	}{
		{
			name:        "no populate corrections",
			corrections: nil,
			want:        false,
		},
		{
			name:        "informational correction only (zone exists or not creatable)",
			corrections: []*models.Correction{{Msg: "nothing to do"}},
			want:        false,
		},
		{
			name:        "pending zone creation (non-nil F)",
			corrections: []*models.Correction{{Msg: "Ensuring zone exists", F: func() error { return nil }}},
			want:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			zone := models.MustNewDomainConfig("example.com")
			zone.StorePopulateCorrections(provider, tc.corrections)
			if got := zoneWillBeCreated(zone, provider); got != tc.want {
				t.Errorf("zoneWillBeCreated() = %v, want %v", got, tc.want)
			}
			// A different provider name must never report a pending creation.
			if zoneWillBeCreated(zone, "other") {
				t.Error("zoneWillBeCreated() reported a creation for an unrelated provider")
			}
		})
	}
}

func Test_reportZonePendingCreation(t *testing.T) {
	const provider = "cf"
	pending := []*models.Correction{{Msg: "Ensuring zone exists", F: func() error { return nil }}}
	none := []*models.Correction{{Msg: "nothing to do"}}

	tests := []struct {
		name              string
		corrections       []*models.Correction
		push              bool
		populateOnPreview bool
		want              bool
	}{
		{name: "preview, zone pending creation", corrections: pending, want: true},
		{name: "push, zone pending creation", corrections: pending, push: true, want: false},
		{name: "preview with --populate-on-preview creates the zone", corrections: pending, populateOnPreview: true, want: false},
		{name: "preview, zone not pending creation", corrections: none, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			zone := models.MustNewDomainConfig("example.com")
			zone.StorePopulateCorrections(provider, tc.corrections)
			if got := reportZonePendingCreation(zone, provider, tc.push, tc.populateOnPreview); got != tc.want {
				t.Errorf("reportZonePendingCreation() = %v, want %v", got, tc.want)
			}
		})
	}
}

// fakeDNSProvider is a models.DNSProvider whose GetNameservers fails when the
// zone does not exist, like providers that look the zone up at the API.
type fakeDNSProvider struct {
	zoneExists bool
	nsCalls    int
}

func (f *fakeDNSProvider) AuditRecords(models.Records) []error { return nil }

func (f *fakeDNSProvider) GetNameservers(string) ([]*models.Nameserver, error) {
	f.nsCalls++
	if !f.zoneExists {
		return nil, errors.New("zone not found")
	}
	return models.ToNameservers([]string{"ns1.example.net", "ns2.example.net"})
}

func (f *fakeDNSProvider) GetZoneRecords(*models.DomainConfig) (models.Records, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeDNSProvider) GetZoneRecordsCorrections(*models.DomainConfig, models.Records) ([]*models.Correction, int, error) {
	return nil, 0, errors.New("not implemented")
}

type fakeRegistrar struct {
	calls int
}

func (f *fakeRegistrar) GetRegistrarCorrections(*models.DomainConfig) ([]*models.Correction, error) {
	f.calls++
	return nil, nil
}

func Test_splitPendingCreation(t *testing.T) {
	zone := models.MustNewDomainConfig("example.com")
	zone.DNSProviderInstances = []*models.DNSProviderInstance{
		{ProviderBase: models.ProviderBase{Name: "new"}},
		{ProviderBase: models.ProviderBase{Name: "old"}},
	}
	zone.StorePopulateCorrections("new", []*models.Correction{{Msg: "Ensuring zone exists", F: func() error { return nil }}})

	tests := []struct {
		name              string
		push              bool
		populateOnPreview bool
		wantPending       int
	}{
		{name: "preview", wantPending: 1},
		{name: "push", push: true, wantPending: 0},
		{name: "preview with --populate-on-preview", populateOnPreview: true, wantPending: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			existing, pending := splitPendingCreation(zone, tc.push, tc.populateOnPreview)
			if len(pending) != tc.wantPending || len(existing)+len(pending) != 2 {
				t.Errorf("splitPendingCreation() = %d existing, %d pending; want %d pending", len(existing), len(pending), tc.wantPending)
			}
			if tc.wantPending == 1 && pending[0].Name != "new" {
				t.Errorf("splitPendingCreation() pending = %q, want %q", pending[0].Name, "new")
			}
		})
	}
}

// Previewing a zone that push will create must not fail just because the
// provider can't list nameservers for a zone that does not exist yet.
func Test_oneZone_pendingCreationSkipsDelegation(t *testing.T) {
	newProv := &fakeDNSProvider{zoneExists: false}
	oldProv := &fakeDNSProvider{zoneExists: true}
	reg := &fakeRegistrar{}

	zone := models.MustNewDomainConfig("example.com")
	zone.RegistrarName = "reg"
	zone.RegistrarInstance = &models.RegistrarInstance{ProviderBase: models.ProviderBase{Name: "reg"}, Driver: reg}
	zone.DNSProviderInstances = []*models.DNSProviderInstance{
		{ProviderBase: models.ProviderBase{Name: "new"}, Driver: newProv, NumberOfNameservers: -1},
		{ProviderBase: models.ProviderBase{Name: "old"}, Driver: oldProv, NumberOfNameservers: -1},
	}
	zone.StorePopulateCorrections("new", []*models.Correction{{Msg: "Ensuring zone exists", F: func() error { return nil }}})

	// Limit record processing to the pending provider; the fake can't diff records.
	if err := oneZone(zone, PPreviewArgs{FilterArgs: FilterArgs{Providers: "new"}}, false); err != nil {
		t.Fatalf("oneZone() error = %v, want nil", err)
	}

	if newProv.nsCalls != 0 {
		t.Errorf("GetNameservers called %d time(s) on the provider where the zone is pending creation", newProv.nsCalls)
	}
	if oldProv.nsCalls != 1 {
		t.Errorf("GetNameservers called %d time(s) on the provider where the zone exists, want 1", oldProv.nsCalls)
	}
	if len(zone.Nameservers) != 2 {
		t.Errorf("zone has %d nameservers, want 2 from the existing provider", len(zone.Nameservers))
	}
	if reg.calls != 0 {
		t.Errorf("GetRegistrarCorrections called %d time(s), want 0", reg.calls)
	}

	corrections := zone.GetCorrections("reg")
	if len(corrections) != 1 || corrections[0].F != nil || !strings.Contains(corrections[0].Msg, "Skipping registrar") {
		t.Errorf("registrar corrections = %+v, want one informational message", corrections)
	}
	if got := zone.GetChangeCount("reg"); got != 0 {
		t.Errorf("registrar change count = %d, want 0", got)
	}
}

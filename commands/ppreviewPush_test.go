package commands

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rtypecontrol"
)

func Test_whichZonesToProcess(t *testing.T) {

	dcNoTag := &models.DomainConfig{Name: "example.com"}
	dcNoTag2 := &models.DomainConfig{Name: "example.net"}
	dcTaggedEmpty := &models.DomainConfig{Name: "example.com!"}
	dcTaggedGeorge := &models.DomainConfig{Name: "example.com!george"}
	dcTaggedJohn := &models.DomainConfig{Name: "example.com!john"}

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
		rtypecontrol.FixLegacyDC(dc)
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
			zone := &models.DomainConfig{Name: "example.com"}
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
			zone := &models.DomainConfig{Name: "example.com"}
			zone.StorePopulateCorrections(provider, tc.corrections)
			if got := reportZonePendingCreation(zone, provider, tc.push, tc.populateOnPreview); got != tc.want {
				t.Errorf("reportZonePendingCreation() = %v, want %v", got, tc.want)
			}
		})
	}
}

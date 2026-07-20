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

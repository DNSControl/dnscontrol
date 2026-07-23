package rcflag_test

import (
	"slices"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/rcflag"
)

func TestProcessForNewRecordConfig(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		args  []any
		want  []any
		want2 rcflag.Flags
	}{
		{"empty", nil, nil, rcflag.Flags{}},
		{"no flags", []any{"one", 2}, []any{"one", 2}, rcflag.Flags{}},
		{"srv weird split", []any{1, rcflag.SrvWeirdSplit, 2}, []any{1, 2}, rcflag.Flags{SrvWeirdSplit: true}},
		{"txt dont parse", []any{rcflag.TxtDontParse, "text"}, []any{"text"}, rcflag.Flags{TxtDontParse: true}},
		{"both flags", []any{rcflag.SrvWeirdSplit, "text", rcflag.TxtDontParse}, []any{"text"}, rcflag.Flags{SrvWeirdSplit: true, TxtDontParse: true}},
		{"duplicate flags", []any{rcflag.SrvWeirdSplit, rcflag.SrvWeirdSplit}, nil, rcflag.Flags{SrvWeirdSplit: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2 := rcflag.ProcessForNewRecordConfig(tt.args)
			if !slices.Equal(got, tt.want) {
				t.Errorf("ProcessForNewRecordConfig() = %v, want %v", got, tt.want)
			}
			if got2 != tt.want2 {
				t.Errorf("ProcessForNewRecordConfig() = %+v, want %+v", got2, tt.want2)
			}
		})
	}
}

func TestProcessForNewRecordConfigParse(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		rcflagList []any
		want       rcflag.Flags
	}{
		{"empty", nil, rcflag.Flags{}},
		{"srv weird split", []any{rcflag.SrvWeirdSplit}, rcflag.Flags{SrvWeirdSplit: true}},
		{"txt dont parse", []any{rcflag.TxtDontParse}, rcflag.Flags{TxtDontParse: true}},
		{"both flags", []any{rcflag.SrvWeirdSplit, rcflag.TxtDontParse}, rcflag.Flags{SrvWeirdSplit: true, TxtDontParse: true}},
		{"duplicate flags", []any{rcflag.SrvWeirdSplit, rcflag.SrvWeirdSplit}, rcflag.Flags{SrvWeirdSplit: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rcflag.ProcessForNewRecordConfigParse(tt.rcflagList)
			if got != tt.want {
				t.Errorf("ProcessForNewRecordConfigParse() = %v, want %v", got, tt.want)
			}
		})
	}
}

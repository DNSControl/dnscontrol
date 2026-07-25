package mustbe_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/mustbe"
)

func TestTxts(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		args []any
		want []string
	}{
		{"simple", []any{"simple"}, []string{"simple"}},
		{"any", []any{1, "foo", 45.2}, []string{"1foo45.2"}},
		{"254", []any{strings.Repeat("a", 254)}, []string{
			strings.Repeat("a", 254),
		}},
		{"255", []any{strings.Repeat("b", 255)}, []string{
			strings.Repeat("b", 255),
		}},
		{"256", []any{strings.Repeat("c", 256)}, []string{
			strings.Repeat("c", 255),
			strings.Repeat("c", 1),
		}},
		{"257", []any{strings.Repeat("d", 257)}, []string{
			strings.Repeat("d", 255),
			strings.Repeat("d", 2),
		}},
		{"513", []any{strings.Repeat("e", 513)}, []string{
			strings.Repeat("e", 255),
			strings.Repeat("e", 255),
			strings.Repeat("e", (513 - 255 - 255)),
		}},
		{"200 200 200", []any{
			strings.Repeat("f", 200),
			strings.Repeat("f", 200),
			strings.Repeat("f", 200),
		}, []string{
			strings.Repeat("f", 255),
			strings.Repeat("f", 255),
			strings.Repeat("f", (600 - 255 - 255)),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustbe.Txts(tt.args...)
			// TODO: update the condition below to compare got with tt.want.
			if !slices.Equal(got, tt.want) {
				t.Errorf("Txts() = %v, want %v", got, tt.want)
			}
		})
	}
}

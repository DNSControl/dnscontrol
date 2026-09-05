package mustbe

import (
	"testing"
)

func TestSoaMailbox(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		arg  any
		want string
	}{
		{"a", "tal.example.com", "tal.example.com"},
		{"a", "DEFAULT_NOT_SET.", "default_not_set."},
		{"b", "tal@example.com", "tal.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SoaMailbox(tt.arg)
			if got != tt.want {
				t.Errorf("SoaMailbox() = %v, want %v", got, tt.want)
			}
		})
	}
}

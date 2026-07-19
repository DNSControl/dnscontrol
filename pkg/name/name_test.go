package name

import "testing"

func TestAddOrigin(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		origin  string
		wantDot string
		wantND  string
	}{
		{name: "relative name", label: "foo", origin: "bar.com", wantDot: "foo.bar.com.", wantND: "foo.bar.com"},
		{name: "apex", label: "@", origin: "bar.com", wantDot: "bar.com.", wantND: "bar.com"},
		{name: "empty label", label: "", origin: "bar.com", wantDot: "bar.com.", wantND: "bar.com"},
		{name: "absolute name", label: "foo.com.", origin: "bar.com", wantDot: "foo.com.", wantND: "foo.com"},
		{name: "trailing dot", label: "foo", origin: "bar.com.", wantDot: "foo.bar.com.", wantND: "foo.bar.com"},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			if got := ToFqdnWithDot(tt.label, tt.origin); got != tt.wantDot {
				t.Fatalf("ToFqdnWithDot(%q, %q) = %q; want %q", tt.label, tt.origin, got, tt.wantDot)
			}
		})

		t.Run(tt.name, func(t *testing.T) {
			if got := ToFqdnNoDot(tt.label, tt.origin); got != tt.wantND {
				t.Fatalf("ToFqdnNoDot(%q, %q) = %q; want %q", tt.label, tt.origin, got, tt.wantND)
			}
		})

	}
}

func TestToShort(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		s      string
		origin string
		want   string
	}{
		{name: "empty name", s: "", origin: "example.com", want: "@"},
		{name: "at sign", s: "@", origin: "example.com", want: "@"},
		{name: "relative name", s: "foo", origin: "example.com", want: "foo"},
		{name: "apex", s: "example.com.", origin: "example.com", want: "@"},
		{name: "name in origin", s: "foo.example.com.", origin: "example.com", want: "foo"},
		{name: "nested name in origin", s: "a.b.example.com.", origin: "example.com", want: "a.b"},
		{name: "name outside origin", s: "foo.other.com.", origin: "example.com", want: "foo.other.com."},
		{name: "similar suffix", s: "bexample.com.", origin: "example.com", want: "bexample.com."},
		{name: "a similar suffix", s: "a.bexample.com.", origin: "example.com", want: "a.bexample.com."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToShort(tt.s, tt.origin)
			if got != tt.want {
				t.Errorf("ToShort() = %v, want %v", got, tt.want)
			}
		})
	}
}

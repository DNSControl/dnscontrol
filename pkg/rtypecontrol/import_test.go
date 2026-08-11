package rtypecontrol

import "testing"

func Test_stutters(t *testing.T) {
	tests := []struct {
		name  string
		rName string
		want  bool
	}{
		{
			name:  "@ symbol should not stutter",
			rName: "@",
			want:  false,
		},
		{
			name:  "exact domain match should stutter",
			rName: "example.com",
			want:  true,
		},
		{
			name:  "subdomain with dot prefix should stutter",
			rName: "www.example.com",
			want:  true,
		},
		{
			name:  "simple subdomain should not stutter",
			rName: "www",
			want:  false,
		},
		{
			name:  "partial match without dot should not stutter",
			rName: "testexample.com",
			want:  false,
		},
		{
			name:  "empty name should not stutter",
			rName: "",
			want:  false,
		},
		{
			name:  "nested subdomain should stutter",
			rName: "api.staging.example.com",
			want:  true,
		},
		{
			name:  "different domain should not stutter",
			rName: "example.org",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stutters(tt.rName, "example.com"); got != tt.want {
				t.Errorf("stutters(%q, %q) = %v, want %v", tt.rName, "example.com", got, tt.want)
			}
		})
	}
}

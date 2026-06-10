package models

import "testing"

func Test_doesStutter(t *testing.T) {
	tests := []struct {
		name  string
		rName string
		want  bool
	}{
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
			name:  "nested subdomain should stutter",
			rName: "api.staging.example.com",
			want:  true,
		},
		{
			name:  "@ symbol should NOT stutter",
			rName: "@",
			want:  false,
		},
		{
			name:  "different domain should NOT stutter",
			rName: "example.org",
			want:  false,
		},
		{
			name:  "simple subdomain should NOT stutter",
			rName: "www",
			want:  false,
		},
		{
			name:  "similar domain should NOT stutter",
			rName: "testexample.com",
			want:  false,
		},
		{
			name:  "empty name should NOT stutter",
			rName: "",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := doesStutter(tt.rName, "example.com"); got != tt.want {
				t.Errorf("stutters(%q, %q) = %v, want %v", tt.rName, "example.com", got, tt.want)
			}
		})
	}
}

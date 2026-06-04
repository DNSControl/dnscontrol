package privatetypes

import (
	"reflect"
	"testing"
)

func TestTokensToArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "single token",
			input:    []string{"one"},
			expected: []string{"one"},
		},
		{
			name:     "two tokens",
			input:    []string{"one", "two"},
			expected: []string{"one", "two"},
		},
		{
			name:     "three tokens",
			input:    []string{"one", "two", "three"},
			expected: []string{"one", "two", "three"},
		},
		{
			name:     "quoted string",
			input:    []string{"\"", "one", "\""},
			expected: []string{"one"},
		},
		{
			name:     "quoted string with following token",
			input:    []string{"\"", "one", "\"", "two"},
			expected: []string{"one", "two"},
		},
		{
			name:     "token before quoted string",
			input:    []string{"one", "\"", "two", "\""},
			expected: []string{"one", "two"},
		},
		{
			name:     "incomplete quoted string at end",
			input:    []string{"one", "\"", "two"},
			expected: []string{"one", "\"", "two"},
		},
		{
			name:     "lone quote at end",
			input:    []string{"one", "\""},
			expected: []string{"one", "\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TokensToArgs(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("TokensToArgs(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

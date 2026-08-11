package txtutil

import (
	"reflect"
	"testing"
)

func Test_splitChunks(t *testing.T) {
	type args struct {
		buf string
		lim int
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"0", args{"", 3}, []string{}},
		{"1", args{"a", 3}, []string{"a"}},
		{"2", args{"ab", 3}, []string{"ab"}},
		{"3", args{"abc", 3}, []string{"abc"}},
		{"4", args{"abcd", 3}, []string{"abc", "d"}},
		{"5", args{"abcde", 3}, []string{"abc", "de"}},
		{"6", args{"abcdef", 3}, []string{"abc", "def"}},
		{"7", args{"abcdefg", 3}, []string{"abc", "def", "g"}},
		{"8", args{"abcdefgh", 3}, []string{"abc", "def", "gh"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitChunks(tt.args.buf, tt.args.lim); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitChunks() = %v, want %v", got, tt.want)
			}
		})
	}
}

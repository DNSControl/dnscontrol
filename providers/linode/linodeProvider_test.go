package linode

import "testing"

func TestFixTTL(t *testing.T) {
	for i, test := range []struct {
		given, expected uint32
	}{
		{0, 0},
		{1, 300},
		{299, 300},
		{300, 300},
		{301, 3600},
		{2419202, 2419200},
		{600, 3600},
		{3600, 3600},
	} {
		found := fixTTL(test.given)
		if found != test.expected {
			t.Errorf("Test %d: Expected %d, but was %d", i, test.expected, found)
		}
	}
}

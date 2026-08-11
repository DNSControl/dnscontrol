package domainnameshop

import "testing"

func TestFixTTL(t *testing.T) {
	for i, test := range []struct {
		given, expected uint32
	}{
		{1, minAllowedTTL},
		{multiplierTTL*5 - 1, multiplierTTL * 4},
		{maxAllowedTTL + 1, maxAllowedTTL},
		{0, 60},
		{59, 60},
		{60, 60},
		{61, 60},
		{119, 60},
		{120, 120},
		{121, 120},
	} {
		found := fixTTL(test.given)
		if found != test.expected {
			t.Errorf("Test %d: Expected %d, but was %d", i, test.expected, found)
		}
	}
}

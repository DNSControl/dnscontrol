package dnstree_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/pkg/dnstree"
)

func Test_domaintree(t *testing.T) {
	t.Run("Single FQDN",
		executeTreeTest(
			[]string{
				"other.example.com",
			},
			[]string{"other.example.com"},
			[]string{"com", "x.example.com", "x.www.example.com", "example.com"},
		),
	)

	t.Run("Wildcard",
		executeTreeTest(
			[]string{
				"*.example.com",
			},
			[]string{"example.com", "other.example.com"},
			[]string{"com", "example.nl", "*.com"},
		),
	)

	t.Run("Combined domains",
		executeTreeTest(
			[]string{
				"*.other.example.com",
				"specific.example.com",
				"specific.example.nl",
			},
			[]string{"any.other.example.com", "specific.example.com", "specific.example.nl"},
			[]string{"com", "nl", "", "example.nl", "other.nl"},
		),
	)
}

func executeTreeTest(inputs []string, founds []string, missings []string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		tree := dnstree.Create[any]()
		for _, input := range inputs {
			tree.Set(input, struct{}{})
		}

		for _, found := range founds {
			if tree.Has(found) == false {
				t.Errorf("Expected %s to be found in tree, but is missing", found)
			}
		}

		for _, missing := range missings {
			if tree.Has(missing) == true {
				t.Errorf("Expected %s to be missing in tree, but is found", missing)
			}
		}
	}
}

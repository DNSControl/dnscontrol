package dnssort_test

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/pkg/dnsgraph"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnsgraph/testutils"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnssort"
)

func Test_graphsort(t *testing.T) {
	t.Run("Direct dependency",
		executeGraphSort(
			[]testutils.StubRecord{
				{NameFQDN: "www.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "example.com"}}},
				{NameFQDN: "example.com", Dependencies: []dnsgraph.Dependency{}},
			},
			[]string{
				"example.com",
				"www.example.com",
			},
			[]string{},
		),
	)

	t.Run("Already in correct order",
		executeGraphSort(
			[]testutils.StubRecord{
				{NameFQDN: "example.com", Dependencies: []dnsgraph.Dependency{}},
				{NameFQDN: "www.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "example.com"}}},
			},
			[]string{
				"example.com",
				"www.example.com",
			},
			[]string{},
		),
	)

	t.Run("Use wildcards",
		executeGraphSort(
			[]testutils.StubRecord{
				{NameFQDN: "www.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "a.test.example.com"}}},
				{NameFQDN: "*.test.example.com", Dependencies: []dnsgraph.Dependency{}},
			},
			[]string{
				"*.test.example.com",
				"www.example.com",
			},
			[]string{},
		),
	)

	t.Run("Cyclic dependency added on the end",
		executeGraphSort(
			[]testutils.StubRecord{
				{NameFQDN: "a.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "b.example.com"}}},
				{NameFQDN: "b.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "a.example.com"}}},
				{NameFQDN: "www.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "example.com"}}},
				{NameFQDN: "example.com", Dependencies: []dnsgraph.Dependency{}},
			},
			[]string{
				"example.com",
				"www.example.com",
				"a.example.com",
				"b.example.com",
			},
			[]string{
				"a.example.com",
				"b.example.com",
			},
		),
	)

	t.Run("Dependency on the same named record resolves in correct order",
		executeGraphSort(
			[]testutils.StubRecord{
				{NameFQDN: "a.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "a.example.com"}}},
				{NameFQDN: "www.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "example.com"}}},
				{NameFQDN: "example.com", Dependencies: []dnsgraph.Dependency{{}}},
				{NameFQDN: "a.example.com", Dependencies: []dnsgraph.Dependency{}},
			},
			[]string{
				"example.com",
				"a.example.com",
				"a.example.com",
				"www.example.com",
			},
			[]string{},
		),
	)

	t.Run("Ignores external dependency",
		executeGraphSort(
			[]testutils.StubRecord{
				{NameFQDN: "mail.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "mail.external.tld"}}},
			},
			[]string{
				"mail.example.com",
			},
			[]string{},
		),
	)

	t.Run("Deletions in correct order",
		executeGraphSort(
			[]testutils.StubRecord{
				{NameFQDN: "mail.example.com", Dependencies: []dnsgraph.Dependency{}},
				{NameFQDN: "example.com", Dependencies: []dnsgraph.Dependency{{NameFQDN: "mail.example.com", Type: dnsgraph.BackwardDependency}}},
			},
			[]string{
				"example.com",
				"mail.example.com",
			},
			[]string{},
		),
	)

	t.Run("A Change with dependency on old and new state",
		executeGraphSort(
			[]testutils.StubRecord{
				{NameFQDN: "bar2.example.com", Dependencies: []dnsgraph.Dependency{{}}},
				{NameFQDN: "foo.example.com", Dependencies: []dnsgraph.Dependency{{NameFQDN: "bar2.example.com", Type: dnsgraph.BackwardDependency}, {NameFQDN: "new2.example.com", Type: dnsgraph.ForwardDependency}}},
				{NameFQDN: "new2.example.com", Dependencies: []dnsgraph.Dependency{{}}},
			},
			[]string{
				"new2.example.com",
				"foo.example.com",
				"bar2.example.com",
			},
			[]string{},
		),
	)

	t.Run("Reports first and ignore dependencies",
		executeGraphSort(
			[]testutils.StubRecord{
				{NameFQDN: "example.com", Dependencies: []dnsgraph.Dependency{}},
				{NameFQDN: "test.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "example.com"}}, Type: dnsgraph.Report},
			},
			[]string{
				"test.example.com",
				"example.com",
			},
			[]string{},
		),
	)

	t.Run("Reports are not resolvable as dependencies",
		executeGraphSort(
			[]testutils.StubRecord{
				{NameFQDN: "test2.example.com", Dependencies: []dnsgraph.Dependency{{Type: dnsgraph.ForwardDependency, NameFQDN: "test.example.com"}}},
				{NameFQDN: "test.example.com", Dependencies: []dnsgraph.Dependency{}, Type: dnsgraph.Report},
			},
			[]string{
				"test.example.com",
				"test2.example.com",
			},
			[]string{},
		),
	)
}

func executeGraphSort(inputOrder []testutils.StubRecord, expectedOutputOrder []string, expectedUnresolved []string) func(*testing.T) {
	return func(t *testing.T) {
		inputSortableRecords := testutils.StubRecordsAsGraphable(inputOrder)
		t.Helper()
		result := dnssort.SortUsingGraph(inputSortableRecords)

		if len(expectedOutputOrder) != len(result.SortedRecords) {
			t.Errorf("Missing records after sort. Expected order: %v. Got order: %v\n", expectedOutputOrder, dnsgraph.GetRecordsNamesForGraphables(result.SortedRecords))
		} else {
			for iX := range expectedOutputOrder {
				if result.SortedRecords[iX].GetName() != expectedOutputOrder[iX] {
					t.Errorf("Invalid order on index %d after sort. Expected order: %v. Got order: %v\n", iX, expectedOutputOrder, dnsgraph.GetRecordsNamesForGraphables(result.SortedRecords))
				}
			}
		}

		if len(expectedUnresolved) != len(result.UnresolvedRecords) {
			t.Errorf("Missing unresolved records. Expected unresolved: %v. Got: %v\n", expectedUnresolved, dnsgraph.GetRecordsNamesForGraphables(result.UnresolvedRecords))
		} else {
			for iX := range expectedUnresolved {
				if result.UnresolvedRecords[iX].GetName() != expectedUnresolved[iX] {
					t.Errorf("Invalid unresolved records after sort. Expected: %v. Got: %v\n", expectedOutputOrder, dnsgraph.GetRecordsNamesForGraphables(result.UnresolvedRecords))
				}
			}
		}
	}
}

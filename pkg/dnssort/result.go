package dnssort

import "github.com/DNSControl/dnscontrol/v4/pkg/dnsgraph"

// SortResult is the result of a sort function.
type SortResult[T dnsgraph.Graphable] struct {
	// SortedRecords is the sorted records.
	SortedRecords []T
	// UnresolvedRecords is the records that could not be resolved.
	UnresolvedRecords []T
}

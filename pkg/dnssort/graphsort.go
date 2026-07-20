package dnssort

import "github.com/DNSControl/dnscontrol/v4/pkg/dnsgraph"

// SortUsingGraph sorts changes based on their dependencies using a directed graph.
// Most changes have dependencies on other changes.
// Either they are depend on already (a backwards dependency) or their new state depend on another record (a forwards dependency) which can be in our change set or an existing record.
// A rough sketch of our sorting algorithm is as follows
//
// while graph has nodes:
//
//	  foreach node in graph:
//
//		  if node has no dependencies:
//		    add node to sortedNodes
//		    remove node from graph
//
// return sortedNodes
//
// The code below also accounts for the existence of cycles by tracking if any nodes were added to the sorted set in the last round.
func SortUsingGraph[T dnsgraph.Graphable](records []T) SortResult[T] {
	sortState := createDirectedSortState(records)

	for sortState.hasWork() {
		for _, node := range sortState.graph.All {
			sortState.hasResolvedLastRound = false

			if hasUnmetDependencies(node) {
				continue
			}

			sortState.hasResolvedLastRound = true
			sortState.addSortedRecord(node.Data)
			sortState.graph.RemoveNode(node)
		}

		if sortState.hasStalled() {
			break
		}
	}

	sortState.finalize()

	return SortResult[T]{
		SortedRecords:     sortState.sortedRecords,
		UnresolvedRecords: sortState.unresolvedRecords,
	}
}

type directedSortState[T dnsgraph.Graphable] struct {
	graph                *dnsgraph.Graph[T]
	sortedRecords        []T
	unresolvedRecords    []T
	hasResolvedLastRound bool
}

func createDirectedSortState[T dnsgraph.Graphable](records []T) directedSortState[T] {
	changes, reportChanges := splitRecordsByType(records)

	graph := dnsgraph.CreateGraph(changes)

	return directedSortState[T]{
		graph:                graph,
		unresolvedRecords:    []T{},
		sortedRecords:        reportChanges,
		hasResolvedLastRound: false,
	}
}

func splitRecordsByType[T dnsgraph.Graphable](records []T) ([]T, []T) {
	var changes []T
	var reports []T

	for _, record := range records {
		switch record.GetType() {
		case dnsgraph.Report:
			reports = append(reports, record)
		case dnsgraph.Change:
			changes = append(changes, record)
		}
	}

	return changes, reports
}

func (sortState *directedSortState[T]) hasWork() bool {
	return len(sortState.graph.All) > 0
}

func (sortState *directedSortState[T]) hasStalled() bool {
	return !sortState.hasResolvedLastRound
}

func (sortState *directedSortState[T]) addSortedRecord(node T) {
	sortState.sortedRecords = append(sortState.sortedRecords, node)
}

func (sortState *directedSortState[T]) finalize() {
	// Add all of the changes remaining in the graph as unresolved and add them at the end of the sorted result to at least include everything
	if len(sortState.graph.All) > 0 {
		for _, unresolved := range sortState.graph.All {
			sortState.addSortedRecord(unresolved.Data)
			sortState.unresolvedRecords = append(sortState.unresolvedRecords, unresolved.Data)
		}
	}
}

func hasUnmetDependencies[T dnsgraph.Graphable](node *dnsgraph.Node[T]) bool {
	for _, edge := range node.Edges {
		if edge.Dependency.Type == dnsgraph.BackwardDependency && edge.Direction == dnsgraph.IncomingEdge {
			return true
		}

		if edge.Dependency.Type == dnsgraph.ForwardDependency && edge.Direction == dnsgraph.OutgoingEdge {
			return true
		}
	}

	return false
}

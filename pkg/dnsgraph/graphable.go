package dnsgraph

// NodeType enumerates the node types.
type NodeType uint8

const (
	// Change is the type of change.
	Change NodeType = iota
	// Report is a Report.
	Report
)

// DependencyType enumerates the dependency types.
type DependencyType uint8

const (
	// ForwardDependency is a forward dependency.
	ForwardDependency DependencyType = iota
	// BackwardDependency is a backwards dependency.
	BackwardDependency
)

// Dependency is a dependency.
type Dependency struct {
	NameFQDN string
	Type     DependencyType
	// CNAME would generate Dependency{NameFQDN: target, OnlyType: ""}
	// because the dependency is on the target.
	//
	// However a DS's dependency is on its own label, not the target.
	// A DS needs to specify that the dependency is on the NS record at the same label.
	// Therefore a DS record returns Dependency{NameFQDN: label, OnlyType: "NS"}
	// to indicate that the dependency is only for NS records.
	// If OnlyType were "", a DS would also depend on the *other* DS records at its
	// label; with two or more DS there, they'd depend on each other and dnsgraph
	// reports a cycle.
	OnlyType string
}

// Graphable is an interface for things that can be in a graph.
type Graphable interface {
	GetType() NodeType
	GetName() string
	// GetRecordType returns the DNS record type (e.g. "NS", "DS") of the node.
	// It is used to satisfy Dependency.OnlyType filters.
	GetRecordType() string
	GetDependencies() []Dependency
}

// GetRecordsNamesForGraphables returns names in a graph.
func GetRecordsNamesForGraphables[T Graphable](graphables []T) []string {
	var names []string

	for _, graphable := range graphables {
		names = append(names, graphable.GetName())
	}

	return names
}

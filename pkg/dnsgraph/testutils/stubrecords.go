package testutils

import "github.com/DNSControl/dnscontrol/v4/pkg/dnsgraph"

// StubRecord stub.
type StubRecord struct {
	NameFQDN     string
	Dependencies []dnsgraph.Dependency
	Type         dnsgraph.NodeType
}

// GetType stub.
func (record StubRecord) GetType() dnsgraph.NodeType {
	return record.Type
}

// GetName stub.
func (record StubRecord) GetName() string {
	return record.NameFQDN
}

// GetDependencies stub.
func (record StubRecord) GetDependencies() []dnsgraph.Dependency {
	return record.Dependencies
}

// StubRecordsAsGraphable stub.
func StubRecordsAsGraphable(records []StubRecord) []dnsgraph.Graphable {
	sortableRecords := make([]dnsgraph.Graphable, len(records))

	for iX := range records {
		sortableRecords[iX] = records[iX]
	}

	return sortableRecords
}

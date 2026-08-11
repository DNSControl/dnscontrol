package diff2

import (
	"log"

	"github.com/DNSControl/dnscontrol/v4/pkg/dnsgraph"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnssort"
)

func orderByDependencies(changes ChangeList) ChangeList {
	if DisableOrdering {
		log.Println("[Info: ordering of the changes has been disabled.]")
		return changes
	}

	a := dnssort.SortUsingGraph(changes)

	if len(a.UnresolvedRecords) > 0 {
		log.Printf("Warning: Found unresolved records %v.\n"+
			"This can indicate a circular dependency, please ensure all targets from given records exist and no circular dependencies exist in the changeset. "+
			"These unresolved records are still added as changes and pushed to the provider, but will cause issues if and when the provider checks the changes.\n"+
			"For more information and how to disable the reordering please consolidate our documentation at https://docs.dnscontrol.org/developer-info/ordering\n",
			dnsgraph.GetRecordsNamesForGraphables(a.UnresolvedRecords),
		)
	}

	return a.SortedRecords
}

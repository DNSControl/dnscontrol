package models

import "fmt"

type RecordBuilderFn func(dc *DomainConfig, ttl uint32, args []any, subdomain string) (Records, error)

var mapBuilderNameToFn = make(map[string]RecordBuilderFn)

// RegisterBuilder registers a fake type that generates one or more RecordConfigs.
func RegisterBuilder(typeName string, genFn RecordBuilderFn) {

	// typenum -> function that runs the generator and returns a list of RecordConfigs.
	if s, exists := mapBuilderNameToFn[typeName]; exists {
		panic(fmt.Sprintf("mapGeneratorNameToFn[%s] already in use by %v", typeName, s))
	}
	mapBuilderNameToFn[typeName] = genFn
}

func IsBuilder(name string) bool {
	_, ok := mapBuilderNameToFn[name]
	return ok
}

func (dc *DomainConfig) runBuilder(typeName string, ttl uint32, args []any, subdomain string) (Records, error) {
	return mapBuilderNameToFn[typeName](dc, ttl, args, subdomain)
}

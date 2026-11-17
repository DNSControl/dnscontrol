package models

import "fmt"

// RecordBuilderFn is a function that generates RecordConfigs for a builder type.
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

// IsBuilder reports whether name is a registered builder.
func IsBuilder(name string) bool {
	_, ok := mapBuilderNameToFn[name]
	return ok
}

func (dc *DomainConfig) runBuilder(typeName string, ttl uint32, args []any, subdomain string) (recs Records, err error) {
	// The args come from dnsconfig.js, so they are user input and must not
	// panic dnscontrol.  A builder that panics on user input is a bug;
	// report it as an error instead of crashing.
	defer func() {
		if r := recover(); r != nil {
			recs, err = nil, fmt.Errorf("panic building %s record (args=%v): %v", typeName, args, r)
		}
	}()
	return mapBuilderNameToFn[typeName](dc, ttl, args, subdomain)
}

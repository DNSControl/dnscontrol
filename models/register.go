package models

import "fmt"

type RecordGeneratorFn func(dc *DomainConfig, ttl uint32, args []any) (Records, error)

var mapGeneratorNameToFn = make(map[string]RecordGeneratorFn)

// RegisterGenerator registers a fake type that generates one or more RecordConfigs.
func RegisterGenerator(typeName string, genFn RecordGeneratorFn) {

	// typenum -> function that runs the generator and returns a list of RecordConfigs.
	if s, exists := mapGeneratorNameToFn[typeName]; exists {
		panic(fmt.Sprintf("mapGeneratorNameToFn[%s] already in use by %v", typeName, s))
	}
	mapGeneratorNameToFn[typeName] = genFn
}

func IsBuilder(name string) bool {
	_, ok := mapGeneratorNameToFn[name]
	return ok
}

func (dc *DomainConfig) runBuilder(typeName string, ttl uint32, args []any) (Records, error) {
	return mapGeneratorNameToFn[typeName](dc, ttl, args)
}

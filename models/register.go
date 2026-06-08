package models

import "fmt"

type RecordGeneratorFn func(origin string, ttl uint32, args []any) (Records, error)

var mapGeneratorNameToFn = make(map[string]RecordGeneratorFn)

// RegisterGenerator registers a fake type that generates one or more RecordConfigs.
func RegisterGenerator(typeName string, genFn RecordGeneratorFn) {

	// UPDATE typenum -> func(args ...any) (RDATA, error) i.e. a function that creates an RDATA struct for the given code point, with fields filled from the given args.
	if s, exists := mapGeneratorNameToFn[typeName]; exists {
		panic(fmt.Sprintf("mapGeneratorNameToFn[%s] already in use by %v", typeName, s))
	}
	mapGeneratorNameToFn[typeName] = genFn
}

func IsGenerator(name string) bool {
	_, ok := mapGeneratorNameToFn[name]
	return ok
}

func ExecuteGenerator(typeName string, origin string, ttl uint32, args []any) (Records, error) {
	return mapGeneratorNameToFn[typeName](origin, ttl, args)
}

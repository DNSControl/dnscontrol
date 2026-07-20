package main

import "log"

func main() {
	if err := generateFeatureMatrix(); err != nil {
		log.Fatal(err)
	}
	if err := generateOwnersFile(); err != nil {
		log.Fatal(err)
	}
	funcs, err := generateFunctionTypes()
	if err != nil {
		log.Fatal(err)
	}
	if err := generateDTSFile(funcs); err != nil {
		log.Fatal(err)
	}
	if err := generateGoreleaserFile(); err != nil {
		log.Fatal(err)
	}
	if err := generateLabelerFile(); err != nil {
		log.Fatal(err)
	}
}

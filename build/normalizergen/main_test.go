package main

import (
	"bytes"
	"os"
	"testing"
)

func TestGeneratedNormalizerIsCurrent(t *testing.T) {
	got, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("../../models/rdata_normalize.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("generated normalizer is out of date; run go generate ./models/")
	}
}

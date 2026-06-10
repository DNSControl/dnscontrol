package models

import (
	"strings"
)

func doesStutter(name, origin string) bool {
	// TODO(tlim): MAYBE: Never return true if last char is "."?
	// TODO(tlim): Panic if called with name == ""?
	if name == "@" {
		return false
	}
	if name == origin || strings.HasSuffix(name, "."+origin) {
		return true
	}
	return false
}

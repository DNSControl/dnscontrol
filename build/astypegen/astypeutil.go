package main

import (
	"fmt"
	"go/format"
	"os"
	"reflect"
	"strings"
)

// goIdent turns a DNS type name into a valid Go identifier by dropping any
// character that is not a letter or digit (e.g. "NSAP-PTR" -> "NSAPPTR",
// "CLOUDFLAREAPI_SINGLE_REDIRECT" -> "CLOUDFLAREAPISINGLEREDIRECT").
func goIdent(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		default:
			return -1
		}
	}, name)
}

// writeGoFile runs the generated source through gofmt (go/format) and writes it
// to path.
// Copied from pkg/privatetypes/types_generate.go.
func writeGoFile(path string, src []byte) error {
	formatted, err := format.Source(src)
	if err != nil {
		_ = os.WriteFile(path, src, 0o644)
		return fmt.Errorf("gofmt %s: %w", path, err)
	}
	return os.WriteFile(path, formatted, 0o644)
}

// StructName returns the Go struct name and package import path of an RDATA value.
// It returns ("", "") for nil or a type with no name.
func StructName(rd any) (name, pkgPath string) {
	rt := reflect.TypeOf(rd)
	if rt == nil {
		return "", ""
	}
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	return rt.Name(), rt.PkgPath()
}

// Namespace maps an RDATA struct's package import path.
// Possible return values are ("privatetypesrdata", true), ("dnsrdatav2", true), or ("", false).
func Namespace(pkgPath string) (alias string, ok bool) {
	switch {
	case strings.HasSuffix(pkgPath, "/pkg/privatetypes/rdata"):
		return "privatetypesrdata", true
	case pkgPath == "codeberg.org/miekg/dns/rdata":
		return "dnsrdatav2", true
	default:
		return "", false
	}
}

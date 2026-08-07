package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
)

// RDtoFields extracts the exported fields from an RD into []any.
// Fileds prefixed with "RT_" are not included, as they are "Runtime" values, not part of the RD itself.
func RDtoFields(s dnsv2.RDATA) ([]any, error) {
	v := reflect.ValueOf(s)

	// Dereference pointer if a pointer to a struct was passed
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
		// } else {
		// 	return nil, fmt.Errorf("expected pointer to struct, got %v", v.Kind())
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct or pointer to struct, got %v", v.Kind())
	}

	t := v.Type()

	var result []any
	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)

		// Skip unexported fields
		if fieldType.PkgPath != "" && !fieldType.Anonymous {
			continue // skip private fields
		}
		if strings.HasPrefix(fieldType.Name, "RT_") {
			continue
		}

		result = append(result, v.Field(i).Interface())
	}

	return result, nil
}

// RDtoFieldsStrings returns the fields of a RDATA struct as strings.
// Floats are restricted to 2 digits after the decimal.
func RDtoFieldsStrings(s dnsv2.RDATA) ([]string, error) {
	items, err := RDtoFields(s)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(items))
	for i, item := range items {
		switch v := item.(type) {
		case string:
			result[i] = v
		case float32, float64:
			result[i] = fmt.Sprintf("%.2f", v)
		default:
			result[i] = fmt.Sprintf("%v", v)
		}
	}
	return result, nil
}

// RDtoFieldsJS returns the fields of a RDATA struct as Javascript-constants.
// Examples: int: `2`; string:  `"mystring"`
// Floats are restricted to 2 digits after the decimal.
func RDtoFieldsJS(s dnsv2.RDATA) ([]string, error) {

	// Special cases:

	switch v := s.(type) {
	case dnsrdatav2.A, dnsrdatav2.AAAA:
		return []string{`"` + v.String() + `"`}, nil
	}

	// General case:

	items, err := RDtoFields(s)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(items))
	for i, item := range items {
		switch v := item.(type) {
		case string:
			// https://stackoverflow.com/questions/51691901
			b, err := json.Marshal(item)
			if err != nil {
				return nil, err
			}
			result[i] = string(b)
		case float32, float64:
			result[i] = fmt.Sprintf("%.2f", v)
		default:
			result[i] = fmt.Sprintf("%v", v)
		}
	}
	return result, nil
}

package models

import "fmt"

func mergeMetas(metas []map[string]any) (map[string]string, error) {

	m := map[string]string{}

	// Fill in the metadata fields.
	for _, meta := range metas {
		for k, v := range meta {
			oldv, exists := m[k]
			if exists {
				return nil, fmt.Errorf("duplicate metadata key %q (%q -- %q)", k, oldv, fmt.Sprintf("%s", v))
			}
			m[k] = fmt.Sprintf("%s", v)
		}
	}

	return m, nil
}

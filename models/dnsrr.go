package models

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// LineString returns the text representation of the resource record, including the label, ttl, type, and fields.
// This may change some day to include metadata and other fields, skip zero TTLs, and more.
func (rc *RecordConfig) LineString() string {
	return fmt.Sprintf("%s %d IN %s %s", rc.Name, rc.TTL, rc.Type, rc.GetRDATA().String())
}

// StringWithMeta returns LineString followed by sorted, quoted metadata.
func (rc *RecordConfig) StringWithMeta() string {
	if len(rc.Metadata) == 0 {
		return rc.LineString()
	}

	var b strings.Builder
	b.WriteString(rc.LineString())
	b.WriteString(" ;")
	for _, k := range slices.Sorted(maps.Keys(rc.Metadata)) {
		fmt.Fprintf(&b, " %s=%s", k, strconv.Quote(rc.Metadata[k]))
	}
	return b.String()
}

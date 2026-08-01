package models

import "fmt"

// LineString returns the text representation of the resource record, including the label, ttl, type, and fields.
// This may change some day to include metadata and other fields, skip zero TTLs, and more.
func (rc *RecordConfig) LineString() string {
	return fmt.Sprintf("%s %d IN %s %s", rc.Name, rc.TTL, rc.Type, rc.GetRDATA().String())
}

package models

import "fmt"

// methods that make RecordConfig meet the dns.RR interface.

// String returns the text representation of the resource record.
func (rc *RecordConfig) String() string {
	return rc.GetTargetCombined()
}

// LineString returns the text representation of the resource record, including the label, ttl, type, and fields.
// This may change some day to include metadata and other fields. Use sparingly.
func (rc *RecordConfig) LineString() string {
	return fmt.Sprintf("%s %d IN %s %s", rc.Name, rc.TTL, rc.Type, rc.GetRDATA().String())
}

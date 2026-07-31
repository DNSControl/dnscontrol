package models

// methods that make RecordConfig meet the dns.RR interface.

// String returns the text representation of the resource record.
func (rc *RecordConfig) xString() string {
	return rc.GetTargetCombined()
}

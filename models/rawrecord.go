package models

// RawRecordConfig stores the user-input from dnsconfig.js for a DNS
// Record.  This is later processed (in Go) to become a RecordConfig.
// NOTE: Only newer rtypes are processed this way.  Eventually the
// legacy types will be converted.
type RawRecordConfig struct {
	Type    string           `json:"type"`
	Args    []any            `json:"args,omitempty"`
	Metas   []map[string]any `json:"metas,omitempty"`
	TTL     uint32           `json:"ttl,omitempty"`
	FilePos string           `json:"filepos"` // Where in the file this record was defined.
	// SubDomain (if non-empty) is the D_EXTEND() subdomain this record was
	// declared under. It is used to rewrite the label and is stored in the
	// resulting RecordConfig.SubDomain. See ImportRawRecords.
	SubDomain string `json:"subdomain,omitempty"`
}

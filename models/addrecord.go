package models

// AddRecordConfig appends rc to the domain's records.
func (dc *DomainConfig) AddRecordConfig(rc *RecordConfig) {
	dc.Records = append(dc.Records, rc)
}

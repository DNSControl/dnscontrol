package models

func (dc *DomainConfig) AddRecordConfig(rc *RecordConfig) {
	dc.Records = append(dc.Records, rc)
}

package recorddb

import "github.com/DNSControl/dnscontrol/v4/models"

// Functions that make it easier to deal with
// a group of records.
//

// RecordDB is a container of many model.RecordConfig.
type RecordDB = struct {
	labelAndTypeMap map[models.RecordKey]struct{}
}

// NewFromRecords creates a RecordDB from a list of model.RecordConfig.
func NewFromRecords(recs models.Records) *RecordDB {
	result := &RecordDB{}

	result.labelAndTypeMap = make(map[models.RecordKey]struct{}, len(recs))
	for _, rec := range recs {
		result.labelAndTypeMap[rec.Key()] = struct{}{}
	}

	return result
}

// ContainsLT returns true if recdb contains rec. Matching is done
// on the record's label and type (i.e. the RecordKey)
// func (recdb RecordDB) ContainsLT(rec *models.RecordConfig) bool {
//	_, ok := recdb.labelAndTypeMap[rec.Key()]
//	return ok
//}

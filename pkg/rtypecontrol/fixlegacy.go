package rtypecontrol

// import (
// 	"github.com/DNSControl/dnscontrol/v4/models"
// )

// // FixLegacyDC populates .F to compenstate for providers that have not been
// // updated to support RecordConfigV2 when creating RecordConfig.
// // It is called anywhere dc.PostProcess() or models.PostProcessRecords() is
// // called.  Those functions can't call it directly because that would cause an
// // import cycle.
// func FixLegacyDC(dc *models.DomainConfig) {
// 	FixLegacyRecords(&dc.Records, dc.Name)
// }

// // FixLegacyRecords populates .F to compenstate for providers that have not been
// // updated to support RecordConfigV2 when creating RecordConfig.
// // It is called anywhere provider.GetZoneRecords() is called. GetZoneRecords()
// // can't call it directly because that would involve modifying every provider.
// // Instead, providers should be fixed to generate records properly.
// func FixLegacyRecords(recs *models.Records, origin string) {
// 	for _, rec := range *recs {
// 		FixLegacyRecord(rec, origin)
// 	}
// }

// // FixLegacyRecord populates .F to compenstate for providers that have not been
// // updated to support RecordConfigV2 when creating RecordConfig.
// func FixLegacyRecord(rec *models.RecordConfig, origin string) {
// 	//fmt.Printf("DEBUG: FixLegacyRecord for %s %s\n", rec.Type, rec.GetTargetField())
// 	// Populate .F if needed: (legacy)
// 	// That is... If rec.F == nil and this is a "modern" type.
// 	if rec.F == nil {
// 		if fixer, ok := Func[rec.Type]; ok {
// 			fixer.CopyFromLegacyFields(rec)
// 		}
// 	}

// 	rec.FixUp(origin) // Hack. Populates .RDATA and .TypeNum if needed.
// }

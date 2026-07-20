package zonerecs

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/rtypecontrol"
)

// CorrectZoneRecords calls both GetZoneRecords, does any
// post-processing, and then calls GetZoneRecordsCorrections.  The
// name sucks because all the good names were taken.
func CorrectZoneRecords(driver models.DNSProvider, dc *models.DomainConfig) ([]*models.Correction, []*models.Correction, int, error) {
	existingRecords, err := driver.GetZoneRecords(dc)
	if err != nil {
		return nil, nil, 0, err
	}
	rtypecontrol.FixLegacyRecords(&existingRecords) // Call this after GetZoneRecords() to fix providers that haven't been updated for RecordConfigV2.

	// downcase
	models.Downcase(existingRecords)
	models.Downcase(dc.Records)
	models.CanonicalizeTargets(existingRecords, dc.Name)
	models.CanonicalizeTargets(dc.Records, dc.Name)

	// Copy dc so that any correction code that wants to
	// modify the records may. For example, if the provider only
	// supports certain TTL values, it will adjust the ones in
	// dc.Records.
	dc, err = dc.Copy()
	if err != nil {
		return nil, nil, 0, err
	}

	// punycode
	if err := dc.Punycode(); err != nil {
		return nil, nil, 0, err
	}
	// FIXME(tlim) It is a waste to PunyCode every iteration.
	// This should be moved to where the JavaScript is processed.

	everything, actualChangeCount, err := driver.GetZoneRecordsCorrections(dc, existingRecords)
	reports, corrections := splitReportsAndCorrections(everything)
	return reports, corrections, actualChangeCount, err
}

func splitReportsAndCorrections(everything []*models.Correction) (reports, corrections []*models.Correction) {
	for i := range everything {
		if everything[i].F == nil {
			reports = append(reports, everything[i])
		} else {
			corrections = append(corrections, everything[i])
		}
	}
	return reports, corrections
}

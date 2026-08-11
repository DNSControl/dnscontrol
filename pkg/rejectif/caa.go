package rejectif

import (
	"errors"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// Keep these in alphabetical order.

// CaaFlagIsNonZero identifies CAA records where tag is no zero.
func CaaFlagIsNonZero(rc *models.RecordConfig) error {
	if rc.CaaFlag != 0 {
		return errors.New("caa flag is non-zero")
	}
	return nil
}

// CaaTargetContainsWhitespace identifies CAA records that have
// whitespace in the target.
// See https://github.com/DNSControl/dnscontrol/issues/1374
func CaaTargetContainsWhitespace(rc *models.RecordConfig) error {
	if strings.ContainsAny(rc.GetTargetField(), " \t\r\n") {
		return errors.New("caa target contains whitespace")
	}
	return nil
}

// CaaHasEmptyTag detects CAA records with empty tags.
func CaaHasEmptyTag(rc *models.RecordConfig) error {
	if rc.CaaTag == "" {
		return errors.New("caa has empty tag")
	}
	return nil
}

// CaaHasEmptyTarget detects CAA records with empty targets.
func CaaHasEmptyTarget(rc *models.RecordConfig) error {
	if rc.GetTargetField() == "" {
		return errors.New("caa has empty target")
	}
	return nil
}

// // CaaTargetHasSemicolon identifies CAA records that contain semicolons.
// func CaaTargetHasSemicolon(rc *models.RecordConfig) error {
// 	if strings.Contains(rc.GetTargetField(), ";") {
// 		return fmt.Errorf("caa target contains semicolon")
// 	}
// 	return nil
// }

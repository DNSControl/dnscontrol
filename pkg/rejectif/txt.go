package rejectif

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// Keep these in alphabetical order.

// TxtHasBackslash audits TXT records for strings that contains one or more backslashes.
func TxtHasBackslash(rc *models.RecordConfig) error {
	if strings.Contains(rc.GetTargetTXTJoined(), `\`) {
		return errors.New("txtstring contains backslashes")
	}
	return nil
}

// TxtHasUnpairedBackslash audits TXT records for strings that contain an odd number of consecutive backslashes.
// Some providers strip single backslashes or convert odd consecutive backslashes to even.
// e.g., "1back\slash" -> "1backslash", "3back\\\slash" -> "3back\\slash".
func TxtHasUnpairedBackslash(rc *models.RecordConfig) error {
	txt := rc.GetTargetTXTJoined()
	i := 0
	for i < len(txt) {
		if txt[i] == '\\' {
			count := 0
			for i < len(txt) && txt[i] == '\\' {
				count++
				i++
			}
			if count%2 == 1 {
				return errors.New("txtstring contains unpaired backslash (odd count)")
			}
		} else {
			i++
		}
	}
	return nil
}

// TxtHasBackticks audits TXT records for strings that contain backticks.
func TxtHasBackticks(rc *models.RecordConfig) error {
	if strings.Contains(rc.GetTargetTXTJoined(), "`") {
		return errors.New("txtstring contains backtick")
	}
	return nil
}

// TxtHasDoubleQuotes audits TXT records for strings that contain doublequotes.
func TxtHasDoubleQuotes(rc *models.RecordConfig) error {
	if strings.Contains(rc.GetTargetTXTJoined(), `"`) {
		return errors.New("txtstring contains doublequotes")
	}
	return nil
}

// TxtHasSemicolon audits TXT records for strings that contain backticks.
func TxtHasSemicolon(rc *models.RecordConfig) error {
	if strings.Contains(rc.GetTargetTXTJoined(), ";") {
		return errors.New("txtstring contains semicolon")
	}
	return nil
}

// TxtHasSingleQuotes audits TXT records for strings that contain single-quotes.
func TxtHasSingleQuotes(rc *models.RecordConfig) error {
	if strings.Contains(rc.GetTargetTXTJoined(), "'") {
		return errors.New("txtstring contains single-quotes")
	}
	return nil
}

// TxtHasTrailingSpace audits TXT records for strings that end with space.
func TxtHasTrailingSpace(rc *models.RecordConfig) error {
	txt := rc.GetTargetTXTJoined()
	if txt != "" && txt[ultimate(txt)] == ' ' {
		return errors.New("txtstring ends with space")
	}
	return nil
}

// TxtHasUnpairedDoubleQuotes audits TXT records for strings that contain unpaired doublequotes.
func TxtHasUnpairedDoubleQuotes(rc *models.RecordConfig) error {
	if strings.Count(rc.GetTargetTXTJoined(), `"`)%2 == 1 {
		return errors.New("txtstring contains unpaired doublequotes")
	}
	return nil
}

// TxtIsEmpty audits TXT records for empty strings.
func TxtIsEmpty(rc *models.RecordConfig) error {
	if len(rc.GetTargetTXTJoined()) == 0 {
		return errors.New("txtstring is empty")
	}
	return nil
}

// TxtLongerThan returns a function that audits TXT records for length
// greater than maxLength.
func TxtLongerThan(maxLength int) func(rc *models.RecordConfig) error {
	return func(rc *models.RecordConfig) error {
		m := maxLength
		if len(rc.GetTargetTXTJoined()) > m {
			return fmt.Errorf("TXT records longer than %d octets (chars)", m)
		}
		return nil
	}
}

// TxtStartsOrEndsWithSpaces audits TXT records that starts or ends with spaces.
func TxtStartsOrEndsWithSpaces(rc *models.RecordConfig) error {
	txt := rc.GetTargetTXTJoined()
	if len(txt) > 0 && (txt[0] == ' ' || txt[len(txt)-1] == ' ') {
		return errors.New("txtstring starts or ends with spaces")
	}
	return nil
}

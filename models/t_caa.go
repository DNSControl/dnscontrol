package models

import (
	"fmt"
	"slices"
	"strconv"
)

// SetTargetCAA sets the CAA fields.
func (rc *RecordConfig) SetTargetCAA(flag uint8, tag string, target string) error {
	rc.CaaTag = tag
	rc.CaaFlag = flag
	if err := rc.SetTarget(target); err != nil {
		return err
	}
	if rc.Type == "" {
		rc.Type = "CAA"
	}
	if rc.Type != "CAA" {
		panic("assertion failed: SetTargetCAA called when .Type is not CAA")
	}

	// Per: https://www.iana.org/assignments/pkix-parameters/pkix-parameters.xhtml#caa-properties excluding reserved tags
	allowedTags := []string{"issue", "issuewild", "iodef", "contactemail", "contactphone", "issuemail", "issuevmc"}
	if !slices.Contains(allowedTags, tag) {
		return fmt.Errorf("CAA tag (%v) is not one of the valid types", tag)
	}

	return nil
}

// SetTargetCAAStrings is like SetTargetCAA but accepts strings.
func (rc *RecordConfig) SetTargetCAAStrings(flag, tag, target string) error {
	i64flag, err := strconv.ParseUint(flag, 10, 8)
	if err != nil {
		return fmt.Errorf("CAA flag does not fit in 8 bits: %w", err)
	}
	return rc.SetTargetCAA(uint8(i64flag), tag, target)
}

// SetTargetCAAString is like SetTargetCAA but accepts one big string.
// Ex: `0 issue "letsencrypt.org"`.
func (rc *RecordConfig) SetTargetCAAString(s string) error {
	part, err := ParseQuotedFields(s)
	if err != nil {
		return err
	}
	if len(part) != 3 {
		return fmt.Errorf("CAA value does not contain 3 fields: (%#v)", s)
	}
	return rc.SetTargetCAAStrings(part[0], part[1], part[2])
}

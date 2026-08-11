package models

// MakeUnknown turns an RecordConfig into an UNKNOWN type.
func MakeUnknown(rc *RecordConfig, rtype string, contents string, origin string) error {
	rc.Type = "UNKNOWN"
	rc.UnknownTypeName = rtype
	rc.target = contents

	return nil
}

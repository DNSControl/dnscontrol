package models

import "fmt"

// MakeUnknown turns an RecordConfig into an UNKNOWN type.
func MakeUnknown(rc *RecordConfig, rtype string, contents string, origin string) error {
	fmt.Printf("DEBUG: !!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n")
	fmt.Printf("DEBUG: !!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n")
	fmt.Printf("DEBUG: !!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n")
	fmt.Printf("DEBUG: !!!!!!!!!!!!!!!!!!!!!!!!!!!!!\n")
	fmt.Printf("DEBUG: Unknown Type! %q %q %q\n", rtype, contents, origin)
	rc.Type = "UNKNOWN"
	rc.UnknownTypeName = rtype
	rc.target = contents

	return nil
}

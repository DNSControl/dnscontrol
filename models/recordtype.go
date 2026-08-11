package models

// ChangeType converts rc to an rc of type newType.  This is only needed when
// converting from one type to another. Do not use this when initializing a new
// record.
//
// Typically this is used to convert an ALIAS to a CNAME, or SPF to TXT. Using
// this function future-proofs the code since eventually such changes will
// require extra steps.
// REFACTOR(tlim): This function should be rewritten as "ChangeTypeToCNAME()"
// which takes the target as a parameter.
func (rc *RecordConfig) ChangeType(newType string, _ string) {

	rc.Type = newType

}

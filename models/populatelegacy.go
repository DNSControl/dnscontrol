package models

/*
This file supports the backwards-compatibility mode while we are converting to
RecordConfig V3.  In particular, it contains the `copyRDtoLegacyFields()`
function, which copies individual fields from rc.rdata to the old, legacy
fields.
*/

import (
	privatetypesrdata "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
)

// copyRDtoLegacyFields copies the fields from rc.rdata to the legacy fields for that record type.
// This will go away when the migration to RecordConfig V3 is complete.
// For newer record types, this function will be a no-op as they have no legacy fields.
func (rc *RecordConfig) copyRDtoLegacyFields() error {
	// Hack to back-fill legacy fields. This will go away eventually.
	switch rd := rc.GetRDATA().(type) {
	case privatetypesrdata.AKAMAITLC:
		rc.AnswerType = rd.AnswerType
	}

	return nil
}

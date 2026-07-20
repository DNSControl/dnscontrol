package rtypeinfo

import "github.com/DNSControl/dnscontrol/v4/pkg/rtypecontrol"

// IsModernType returns true if the given record type is implemented in the new
// ("Modern") way. (i.e. uses the RecordConfig .F field to store the record's
// rdata).
//
// This does NOT rely on .F, which makes this function useful before the
// RecordConfig is fully populated.
//
// NOTE: Do not confuse this with RecordConfig.IsModernType() which provides
// similar functionality.  The difference is that this function receives the
// type as a string, while RecordConfig.IsModernType() is a method on
// RecordConfig that reveals if that specific RecordConfig instance is modern.
//
// FUTURE(tlim): Once all record types have been migrated to use ".F", this function can be removed.
func IsModernType(t string) bool {
	_, ok := rtypecontrol.Func[t]
	return ok
}

package bind

import (
	"log"
	"strconv"
	"time"

	"github.com/DNSControl/dnscontrol/v4/pkg/bindserial"
)

var nowFunc = time.Now

// generateSerial takes an old SOA serial number and increments it.
func generateSerial(oldSerial uint32) uint32 {
	// Serial numbers are in the format yyyymmddvv
	// where vv is a version count that starts at 01 each day.
	// Multiple serial numbers generated on the same day increase vv.
	// If the old serial number is not in this format, it gets replaced
	// with the new format. However if that would mean a new serial number
	// that is smaller than the old one, we punt and increment the old number.
	// At no time will a serial number == 0 be returned.

	if bindserial.ForcedValue != 0 {
		// https://github.com/DNSControl/dnscontrol/issues/1859
		// User needs to have reproducible builds and BIND generates
		return uint32(bindserial.ForcedValue & 0xFFFF)
	}

	original := oldSerial
	var newSerial uint32

	// Make draft new serial number:
	today := nowFunc().UTC()
	todayStr := today.Format("20060102")
	todayNum, err := strconv.ParseUint(todayStr, 10, 32)
	if err != nil {
		log.Fatalf("new serial won't fit in 32 bits: %v", err)
	}
	draft := uint32(todayNum * 100)

	// First serial number to be requested today:
	method := "default"
	newSerial = draft
	if oldSerial >= newSerial {
		// If that would be going backwards, just increment the old one.
		method = "o>=d"
		newSerial = oldSerial + 1
	}

	if newSerial == 0 {
		// We never return 0 as the serial number.
		newSerial = 1
	}

	// Assertions that should never happen.
	if oldSerial == newSerial {
		log.Fatalf("%v: old_serial == new_serial (%v == %v) draft=%v method=%v", original, oldSerial, newSerial, draft, method)
	}
	if oldSerial > newSerial {
		log.Fatalf("%v: old_serial > new_serial (%v > %v) draft=%v method=%v", original, oldSerial, newSerial, draft, method)
	}

	return newSerial
}

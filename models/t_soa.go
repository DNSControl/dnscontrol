package models

import (
	"fmt"
	"strconv"
	"strings"
)

/*

Providers are not expected to support this record.

Most providers do not support SOA records. They generate them
dynamically behind the scenes.  Providers like BIND (which is
software, not SaaS), must handle SOA records and emulate the dynamic
work that providers do.

*/

// SetTargetSOA sets the SOA fields.
func (rc *RecordConfig) SetTargetSOA(ns, mbox string, serial, refresh, retry, expire, minttl uint32) error {
	// The NS field is stored as the .Target
	if err := rc.SetTarget(ns); err != nil {
		return err
	}
	rc.SoaMbox = mbox
	rc.SoaSerial = serial
	rc.SoaRefresh = refresh
	rc.SoaRetry = retry
	rc.SoaExpire = expire
	rc.SoaMinttl = minttl

	if rc.Type == "" {
		rc.Type = "SOA"
	}
	if rc.Type != "SOA" {
		panic("assertion failed: SetTargetSOA called when .Type is not SOA")
	}

	return nil
}

// SetTargetSOAStrings is like SetTargetSOA but accepts strings.
func (rc *RecordConfig) SetTargetSOAStrings(ns, mbox, serial, refresh, retry, expire, minttl string) error {
	u32serial, err := strconv.ParseUint(serial, 10, 32)
	if err != nil {
		return fmt.Errorf("SOA serial '%v' is invalid: %w", serial, err)
	}

	u32refresh, err := strconv.ParseUint(refresh, 10, 32)
	if err != nil {
		return fmt.Errorf("SOA refresh '%v' is invalid: %w", refresh, err)
	}

	u32retry, err := strconv.ParseUint(retry, 10, 32)
	if err != nil {
		return fmt.Errorf("SOA retry '%v' is invalid: %w", retry, err)
	}

	u32expire, err := strconv.ParseUint(expire, 10, 32)
	if err != nil {
		return fmt.Errorf("SOA expire '%v' is invalid: %w", expire, err)
	}

	u32minttl, err := strconv.ParseUint(minttl, 10, 32)
	if err != nil {
		return fmt.Errorf("SOA minttl '%v' is invalid: %w", minttl, err)
	}

	return rc.SetTargetSOA(ns, mbox, uint32(u32serial), uint32(u32refresh), uint32(u32retry), uint32(u32expire), uint32(u32minttl))
}

// SetTargetSOAString is like SetTargetSOA but accepts one big string.
func (rc *RecordConfig) SetTargetSOAString(s string) error {
	part := strings.Fields(s)
	if len(part) != 7 {
		return fmt.Errorf("SOA value does not contain 7 fields: (%#v)", s)
	}
	return rc.SetTargetSOAStrings(part[0], part[1], part[2], part[3], part[4], part[5], part[6])
}

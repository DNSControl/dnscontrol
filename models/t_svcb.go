package models

import (
	"fmt"
	"strings"

	dnsv1 "github.com/miekg/dns"
)

func (rc *RecordConfig) targetCombinedSVCBRaw() string {
	if rc.SvcParams == "" {
		return fmt.Sprintf("%d %s", rc.SvcPriority, rc.target)
	}
	return fmt.Sprintf("%d %s %s", rc.SvcPriority, rc.target, rc.SvcParams)
}

// SetTargetSVCB sets the SVCB fields.
func (rc *RecordConfig) SetTargetSVCB(priority uint16, target string, params []dnsv1.SVCBKeyValue) error {
	rc.SvcPriority = priority
	if err := rc.SetTarget(target); err != nil {
		return err
	}
	paramsStr := []string{}
	for _, kv := range params {
		paramsStr = append(paramsStr, fmt.Sprintf("%s=%s", kv.Key(), kv.String()))
	}
	rc.SvcParams = strings.Join(paramsStr, " ")
	if rc.Type == "" {
		rc.Type = "SVCB"
	}
	if rc.Type != "SVCB" && rc.Type != "HTTPS" {
		panic("assertion failed: SetTargetSVCB called when .Type is not SVCB or HTTPS")
	}
	return nil
}

// SetTargetSVCBString is like SetTargetSVCB but accepts one big string and the origin so parsing can be done using miekg/dns.
func (rc *RecordConfig) SetTargetSVCBString(origin, contents string) error {
	if rc.Type == "" {
		rc.Type = "SVCB"
	}
	record, err := dnsv1.NewRR(fmt.Sprintf("%s. %s %s", origin, rc.Type, contents))
	if err != nil {
		return fmt.Errorf("could not parse SVCB record: %w", err)
	}
	switch r := record.(type) {
	case *dnsv1.HTTPS:
		return rc.SetTargetSVCB(r.Priority, r.Target, r.Value)
	case *dnsv1.SVCB:
		return rc.SetTargetSVCB(r.Priority, r.Target, r.Value)
	}
	return nil
}

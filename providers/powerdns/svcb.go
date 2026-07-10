package powerdns

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// contentHasPowerDNSSVCBAutoHints reports whether SVCB/HTTPS rdata contains
// PowerDNS's provider-specific automatic address hinting parameters.
func contentHasPowerDNSSVCBAutoHints(content string) bool {
	for field := range strings.FieldsSeq(content) {
		if field == "ipv4hint=auto" || field == "ipv6hint=auto" {
			return true
		}
	}
	return false
}

// recordHasPowerDNSSVCBAutoHints reports whether a RecordConfig is an
// SVCB/HTTPS record using PowerDNS automatic address hinting.
func recordHasPowerDNSSVCBAutoHints(rc *models.RecordConfig) bool {
	if rc.Type != "SVCB" && rc.Type != "HTTPS" {
		return false
	}
	return contentHasPowerDNSSVCBAutoHints(rc.SvcParams)
}

// powerDNSTargetCombined returns PowerDNS API content for a RecordConfig,
// preserving provider-specific SVCB/HTTPS auto hint values that miekg/dns
// cannot parse.
func powerDNSTargetCombined(rc *models.RecordConfig) string {
	if recordHasPowerDNSSVCBAutoHints(rc) {
		if rc.SvcParams == "" {
			return fmt.Sprintf("%d %s", rc.SvcPriority, rc.GetTargetField())
		}
		return fmt.Sprintf("%d %s %s", rc.SvcPriority, rc.GetTargetField(), rc.SvcParams)
	}
	return rc.GetTargetCombined()
}

// rejectPowerDNSSVCBAutoHintsUnsorted rejects PowerDNS auto hint records with
// params that are not sorted by SvcParamKey number.
func rejectPowerDNSSVCBAutoHintsUnsorted(rc *models.RecordConfig) error {
	if !recordHasPowerDNSSVCBAutoHints(rc) {
		return nil
	}

	lastOrder := -1
	for field := range strings.FieldsSeq(rc.SvcParams) {
		order := powerDNSSVCBParamOrder(field)
		if order == -1 {
			continue
		}
		if order < lastOrder {
			return errors.New("PowerDNS SVCB/HTTPS auto hint params must be sorted by SvcParamKey number; ipv4hint must appear before ipv6hint")
		}
		lastOrder = order
	}
	return nil
}

// powerDNSSVCBParamOrder returns the SvcParamKey number for known SVCB params.
func powerDNSSVCBParamOrder(field string) int {
	key := field
	if before, _, ok := strings.Cut(field, "="); ok {
		key = before
	}

	switch key {
	case "mandatory":
		return 0
	case "alpn":
		return 1
	case "no-default-alpn":
		return 2
	case "port":
		return 3
	case "ipv4hint":
		return 4
	case "ech":
		return 5
	case "ipv6hint":
		return 6
	case "dohpath":
		return 7
	case "ohttp":
		return 8
	default:
		return -1
	}
}

package models

import (
	dnsv2 "codeberg.org/miekg/dns"
)

// SetTargetSSHFP sets the SSHFP fields.
func (rc *RecordConfig) SetTargetSSHFP(algorithm uint8, fingerprint uint8, target string) error {
	return legacySetTargetArgs(rc, dnsv2.TypeSSHFP, algorithm, fingerprint, target)
}

// SetTargetSSHFPStrings is like SetTargetSSHFP but accepts strings.
func (rc *RecordConfig) SetTargetSSHFPStrings(algorithm, fingerprint, target string) error {
	return legacySetTargetArgs(rc, dnsv2.TypeSSHFP, algorithm, fingerprint, target)
}

// SetTargetSSHFPString is like SetTargetSSHFP but accepts one big string.
func (rc *RecordConfig) SetTargetSSHFPString(s string) error {
	return legacySetTargetParse(rc, dnsv2.TypeSSHFP, s)
}

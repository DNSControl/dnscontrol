package models

import (
	"fmt"
	"strconv"
	"strings"
)

// SetTargetSSHFP sets the SSHFP fields.
func (rc *RecordConfig) SetTargetSSHFP(algorithm uint8, fingerprint uint8, target string) error {
	rc.SshfpAlgorithm = algorithm
	rc.SshfpFingerprint = fingerprint
	if err := rc.SetTarget(target); err != nil {
		return err
	}
	if rc.Type == "" {
		rc.Type = "SSHFP"
	}
	if rc.Type != "SSHFP" {
		panic("assertion failed: SetTargetSSHFP called when .Type is not SSHFP")
	}

	if algorithm < 1 && algorithm > 4 {
		return fmt.Errorf("SSHFP algorithm (%v) is not one of 1, 2, 3 or 4", algorithm)
	}
	if fingerprint < 1 && fingerprint > 2 {
		return fmt.Errorf("SSHFP fingerprint (%v) is not one of 1 or 2", fingerprint)
	}

	return nil
}

// SetTargetSSHFPStrings is like SetTargetSSHFP but accepts strings.
func (rc *RecordConfig) SetTargetSSHFPStrings(algorithm, fingerprint, target string) error {
	i64algorithm, err := strconv.ParseUint(algorithm, 10, 8)
	if err != nil {
		return fmt.Errorf("SSHFP algorithm does not fit in 8 bits: %w", err)
	}
	i64fingerprint, err := strconv.ParseUint(fingerprint, 10, 8)
	if err != nil {
		return fmt.Errorf("SSHFP fingerprint does not fit in 8 bits: %w", err)
	}
	return rc.SetTargetSSHFP(uint8(i64algorithm), uint8(i64fingerprint), target)
}

// SetTargetSSHFPString is like SetTargetSSHFP but accepts one big string.
func (rc *RecordConfig) SetTargetSSHFPString(s string) error {
	part := strings.Fields(s)
	if len(part) != 3 {
		return fmt.Errorf("SSHFP value does not contain 3 fields: (%#v)", s)
	}
	return rc.SetTargetSSHFPStrings(part[0], part[1], StripQuotes(part[2]))
}

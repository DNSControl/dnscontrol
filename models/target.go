package models

import (
	"fmt"
	"net/netip"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

// GetTargetField returns the "target" field. That is, the field in RDATA that is a hostname.
// We hard-code certain types and others we guess by picking the last field in the struct.
// NOTE: Deprecated. No new code should use this. Get the field you need instead.
func (rc *RecordConfig) GetTargetField() string {

	switch rc.Type {
	case "TXT":
		return rc.GetTargetTXTJoined()
	case "R53_ALIAS":
		// R53_ALIAS's target (DNSName) is not the last field of the RDATA.
		return rc.AsR53ALIAS().Target
	}

	// Return the last field. Not perfect, but good enough until we get rid of this function.
	fx, err := RDtoFieldsStrings(rc.GetRDATA())
	if err != nil {
		return rc.GetRDATA().String()
	}
	if len(fx) == 0 {
		return ""
	}
	return fx[len(fx)-1]
}

// GetTargetIP returns the net.IP stored in .target.
// NOTE: Deprecated. No new code should use this. Use rc.AsA().Addr or rc.AsAAAA().Addr.
func (rc *RecordConfig) GetTargetIP() netip.Addr {
	switch f := rc.GetRDATA().(type) {
	case dnsrdatav2.A:
		return f.Addr
	case dnsrdatav2.AAAA:
		return f.Addr
	}
	panic(fmt.Sprintf("wrong type GetTargetIP(%T)", rc.GetRDATA()))
}

// SetTargetIP sets the target to an IP, verifying this is an appropriate rtype.
// NOTE: Deprecated. No new code should use this.
func (rc *RecordConfig) SetTargetIP(ip netip.Addr) error {
	// TODO(tlim): Verify the rtype is appropriate for an IP.
	//return rc.SetTarget(ip.String())
	switch rc.TypeNum {
	case dnsv2.TypeA:
		rd, err := MakeA("", nil, nrc.Flags{}, ip)
		if err != nil {
			return err
		}
		rc.SetRDATA(rd)
		return nil
	case dnsv2.TypeAAAA:
		rd, err := MakeAAAA("", nil, nrc.Flags{}, ip)
		if err != nil {
			return err
		}
		rc.SetRDATA(rd)
		return nil
	}
	return fmt.Errorf("invalid IP %v", ip)
}

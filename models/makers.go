package models

// Make*() functions for built-in types.
// TODO(tlim): Autogenerate this.  use mustbe.* while we're at it.

import (
	"fmt"
	"slices"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	svcbv2 "codeberg.org/miekg/dns/svcb"

	"github.com/DNSControl/dnscontrol/v5/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	_ "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
	dnsv1 "github.com/miekg/dns"
)

func init() {
	// Register the Maker*() function for public types.
	privatetypes.RegisterMaker(dnsv2.TypeA, MakeA)
	privatetypes.RegisterMaker(dnsv2.TypeAAAA, MakeAAAA)
	privatetypes.RegisterMaker(dnsv2.TypeCAA, MakeCAA)
	privatetypes.RegisterMaker(dnsv2.TypeCNAME, MakeCNAME)
	privatetypes.RegisterMaker(dnsv2.TypeDHCID, MakeDHCID)
	privatetypes.RegisterMaker(dnsv2.TypeDNAME, MakeDNAME)
	privatetypes.RegisterMaker(dnsv2.TypeDNSKEY, MakeDNSKEY)
	privatetypes.RegisterMaker(dnsv2.TypeDS, MakeDS)
	privatetypes.RegisterMaker(dnsv2.TypeHTTPS, MakeHTTPS)
	privatetypes.RegisterMaker(dnsv2.TypeLOC, MakeLOC)
	privatetypes.RegisterMaker(dnsv2.TypeMX, MakeMX)
	privatetypes.RegisterMaker(dnsv2.TypeNAPTR, MakeNAPTR)
	privatetypes.RegisterMaker(dnsv2.TypeNS, MakeNS)
	privatetypes.RegisterMaker(dnsv2.TypeOPENPGPKEY, MakeOPENPGPKEY)
	privatetypes.RegisterMaker(dnsv2.TypePTR, MakePTR)
	privatetypes.RegisterMaker(dnsv2.TypeRP, MakeRP)
	privatetypes.RegisterMaker(dnsv2.TypeSMIMEA, MakeSMIMEA)
	privatetypes.RegisterMaker(dnsv2.TypeSOA, MakeSOA)
	privatetypes.RegisterMaker(dnsv2.TypeSRV, MakeSRV)
	privatetypes.RegisterMaker(dnsv2.TypeSSHFP, MakeSSHFP)
	privatetypes.RegisterMaker(dnsv2.TypeSVCB, MakeSVCB)
	privatetypes.RegisterMaker(dnsv2.TypeTLSA, MakeTLSA)
	privatetypes.RegisterMaker(dnsv2.TypeTXT, MakeTXT)
}

func MakeA(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeA expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	target := args[0]
	return dnsrdatav2.A{Addr: mustbe.IPv4(target)}, nil
}

func MakeALIAS(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeALIAS expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return privatetypesrdata.ALIAS{Target: mustbe.TargetHost(origin, isEnabled, args[0])}, nil
}
func MakeAAAA(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeAAAA expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.AAAA{Addr: mustbe.IPv6(args[0])}, nil
}

func MakeCAA(origin string, metadata map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 2 && len(args) != 3 {
		return nil, fmt.Errorf("MakeCAA expects 2 or 3 arguments, got %d: %+v", len(args), args)
	}
	if len(args) == 2 {
		var flag any = uint8(0)
		if cf, ok := metadata["caaflag"]; ok {
			flag = cf
		}
		return dnsrdatav2.CAA{Flag: mustbe.Uint8(flag), Tag: mustbe.RawString(args[0]), Value: mustbe.RawString(args[1])}, nil
	}

	tag := mustbe.RawString(args[1])
	allowedTags := []string{"issue", "issuewild", "iodef", "contactemail", "contactphone", "issuemail", "issuevmc"}
	if !slices.Contains(allowedTags, tag) {
		return nil, fmt.Errorf("CAA tag (%v) is not one of the valid types", tag)
	}

	return dnsrdatav2.CAA{Flag: mustbe.Uint8(args[0]), Tag: tag, Value: mustbe.RawString(args[2])}, nil

}
func MakeCNAME(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeCNAME expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.CNAME{Target: mustbe.TargetHost(origin, isEnabled, args[0])}, nil
}

func MakeDHCID(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeDHCID expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.DHCID{Digest: mustbe.RawString(args[0])}, nil
}
func MakeDNAME(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeDNAME expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.DNAME{Target: mustbe.TargetHost(origin, isEnabled, args[0])}, nil
}
func MakeDNSKEY(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 4 {
		return nil, fmt.Errorf("MakeDNSKEY expects exactly 4 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.DNSKEY{
		Flags:     mustbe.Uint16(args[0]),
		Protocol:  mustbe.Uint8(args[1]),
		Algorithm: mustbe.Uint8(args[2]),
		PublicKey: mustbe.RawString(args[3]),
		//Tag:       mustbe.Uint16(args[4]),
	}, nil
}

func MakeDS(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 4 {
		return nil, fmt.Errorf("MakeDS expects exactly 4 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.DS{KeyTag: mustbe.Uint16(args[0]), Algorithm: mustbe.Uint8(args[1]), DigestType: mustbe.Uint8(args[2]), Digest: mustbe.ToUpperRawString(args[3])}, nil
}

func MakeHTTPS(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 3 {
		return nil, fmt.Errorf("MakeHTTPS expects exactly 3 arguments, got %d: %+v", len(args), args)
	}
	return MakeSVCB(origin, nil, isEnabled, args[0], args[1], args[2])
}

func MakeLOC(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	if len(args) != 7 && len(args) != 12 {
		return nil, fmt.Errorf("MakeLOC expects either 7 or 12 arguments, got %d: %+v", len(args), args)
	}

	if len(args) == 7 {
		return dnsrdatav2.LOC{
			Version:   mustbe.Uint8(args[0]),
			Size:      mustbe.Uint8(args[1]),
			HorizPre:  mustbe.Uint8(args[2]),
			VertPre:   mustbe.Uint8(args[3]),
			Latitude:  mustbe.Uint32(args[4]),
			Longitude: mustbe.Uint32(args[5]),
			Altitude:  mustbe.Uint32(args[6]),
		}, nil
	}

	rc := &RecordConfig{}
	// TODO(tlim): Recfactor SetLOCParams to produce its results outside of an RC.
	rc.SetLOCParams(
		mustbe.Uint8(args[0]),
		mustbe.Uint8(args[1]),
		mustbe.Float32(args[2]),
		mustbe.RawString(args[3]),
		mustbe.Uint8(args[4]),
		mustbe.Uint8(args[5]),
		mustbe.Float32(args[6]),
		mustbe.RawString(args[7]),
		mustbe.Float64(args[8]),
		mustbe.Float32(args[9]),
		mustbe.Float32(args[10]),
		mustbe.Float32(args[11]),
	)

	return dnsrdatav2.LOC{
		Version:   rc.LocVersion,
		Size:      rc.LocSize,
		HorizPre:  rc.LocHorizPre,
		VertPre:   rc.LocVertPre,
		Latitude:  rc.LocLatitude,
		Longitude: rc.LocLongitude,
		Altitude:  rc.LocAltitude,
	}, nil
}

func MakeMIKROTIKFWD(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	return privatetypesrdata.MIKROTIKFWD{ForwardTo: mustbe.TargetHost(origin, isEnabled, args[0])}, nil
}
func MakeMIKROTIKNXDOMAIN(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	return privatetypesrdata.MIKROTIKNXDOMAIN{}, nil
}
func MakeMX(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 2 {
		return nil, fmt.Errorf("MakeMX expects exactly 2 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.MX{Preference: mustbe.Uint16(args[0]), Mx: mustbe.TargetHost(origin, isEnabled, args[1])}, nil
}

func MakeNS(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeNS expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.NS{Ns: mustbe.TargetHost(origin, isEnabled, args[0])}, nil
}
func MakeNAPTR(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 6 {
		return nil, fmt.Errorf("MakeNAPTR expects exactly 6 arguments, got %d: %+v", len(args), args)
	}
	// The NAPTR replacement is a domain name; an empty replacement is canonically
	// the root, ".". Normalize so it renders consistently in zone files.
	replacement := mustbe.RawString(args[5])
	if replacement == "" {
		replacement = "."
	}
	return dnsrdatav2.NAPTR{
		Order:       mustbe.Uint16(args[0]),
		Preference:  mustbe.Uint16(args[1]),
		Flags:       mustbe.RawString(args[2]),
		Service:     mustbe.RawString(args[3]),
		Regexp:      mustbe.RawString(args[4]),
		Replacement: replacement,
	}, nil
}

func MakeOPENPGPKEY(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeOPENPGPKEY expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.OPENPGPKEY{PublicKey: mustbe.OpenPGPKey(args[0])}, nil
}

func MakePORKBUNURLFWD(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	return privatetypesrdata.PORKBUNURLFWD{}, nil
}

func MakePTR(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakePTR expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.PTR{Ptr: mustbe.TargetHost(origin, isEnabled, args[0])}, nil
}

func MakeRP(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 2 {
		return nil, fmt.Errorf("MakeRP expects exactly 2 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.RP{Mbox: mustbe.TargetHost(origin, isEnabled, args[0]), Txt: mustbe.TargetHost(origin, isEnabled, args[1])}, nil
}

func MakeR53ALIAS(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 5 {
		return nil, fmt.Errorf("MakeR53ALIAS expects exactly 5 arguments, got %d: %+v", len(args), args)
	}
	return privatetypesrdata.R53ALIAS{
		AliasType: mustbe.RawString(args[0]),
		Target:    mustbe.TargetHost(origin, isEnabled, args[1]),
		// ZoneID:           mustbe.RawString(args[2]),
		// EvalTargetHealth: mustbe.RawString(args[3]),
		// FIXME(tlim): EvalTargetHealth is a boolean in our internal model but the R53ALIAS type expects a string. This is a hack to convert it to the expected format. We should probably change the R53ALIAS type to use a boolean for this field.
	}, nil
}

func MakeSMIMEA(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 4 {
		return nil, fmt.Errorf("MakeSMIMEA expects exactly 4 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.SMIMEA{Usage: mustbe.Uint8(args[0]), Selector: mustbe.Uint8(args[1]), MatchingType: mustbe.Uint8(args[2]), Certificate: mustbe.RawString(args[3])}, nil
}

func MakeSOA(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 6 && len(args) != 7 {
		return nil, fmt.Errorf("MakeSOA expects exactly 6 or 7 arguments, got %d: %+v", len(args), args)
	}
	// dnsconfig.js's SOA() passes 6 args (no serial): the user can not specify
	// the serial number, it is managed by DNSControl. Re-deriving the RDATA from
	// an existing RecordConfig (FixUp) or parsing a zone (e.g. BIND) passes 7
	// args, with the serial at args[2].
	// The one exception is that for hermetic builds/tests, "dnscontrol preview --bindserial" exists.
	var serial any = uint32(0)
	rest := args[2:]
	if len(args) == 7 {
		serial = args[2]
		rest = args[3:]
	}
	return dnsrdatav2.SOA{
		Ns:      mustbe.TargetHost(origin, isEnabled, args[0]),
		Mbox:    mustbe.SoaMailbox(args[1]),
		Serial:  mustbe.Uint32(serial),
		Refresh: mustbe.Uint32(rest[0]),
		Retry:   mustbe.Uint32(rest[1]),
		Expire:  mustbe.Uint32(rest[2]),
		Minttl:  mustbe.Uint32(rest[3]),
	}, nil
}

func MakeSRV(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 4 {
		return nil, fmt.Errorf("MakeSRV expects exactly 4 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.SRV{Priority: mustbe.Uint16(args[0]), Weight: mustbe.Uint16(args[1]), Port: mustbe.Uint16(args[2]), Target: mustbe.TargetHostSRV(origin, isEnabled, args[3])}, nil
}

func MakeSSHFP(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 3 {
		return nil, fmt.Errorf("MakeSSHFP expects exactly 3 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.SSHFP{Algorithm: mustbe.Uint8(args[0]), Type: mustbe.Uint8(args[1]), FingerPrint: mustbe.ToUpperRawString(args[2])}, nil
}

func MakeSVCB(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	// args can be a string (which we parse), a []dnsv1.SVCBKeyValue or a []svcbv2.Pair.
	// If it's a string, this is where we turn `ech=IGNORE` into `ech=1000`.
	if len(args) != 3 {
		return nil, fmt.Errorf("MakeSVCB expects exactly 3 arguments, got %d: %+v", len(args), args)
	}
	priority := args[0]
	target := args[1]
	params := args[2]

	if priority == 0 {
		return dnsrdatav2.SVCB{Priority: mustbe.Uint16(priority), Target: mustbe.TargetHost(origin, isEnabled, target)}, nil
	}

	switch v := params.(type) {
	case []dnsv1.SVCBKeyValue:
		pv2, err := convertSVCBv1v2(v) // This hasn't been tested much.
		if err != nil {
			panic("BUG: Failed to convert SVCB parameters from v1 to v2: " + err.Error())
		}
		return dnsrdatav2.SVCB{Priority: mustbe.Uint16(priority), Target: mustbe.TargetHost(origin, isEnabled, target), Value: pv2}, nil
	case []svcbv2.Pair:
		return dnsrdatav2.SVCB{Priority: mustbe.Uint16(priority), Target: mustbe.TargetHost(origin, isEnabled, target), Value: v}, nil
	case string:
		// ech=IGNORE is special. It means "take the ech value from the existing
		// record".  We replace it with the byte sequence 0x10 0x00 here. Later,
		// we look for that value and replace it with the existing record's
		// value. This works because we can assume there are no ech= values that
		// are actually 0x10 0x00.  (The ech=IGNORE value is stored as ech=1000
		// in the wire format.)
		v = strings.ReplaceAll(v, "IGNORE", "1000")

		pairs, err := stringToSvcbv2Values(origin, v)
		if err != nil {
			return nil, err
		}
		return dnsrdatav2.SVCB{Priority: mustbe.Uint16(priority), Target: mustbe.TargetHost(origin, isEnabled, target), Value: pairs}, nil

	}

	panic(fmt.Sprintf("BUG: Invalid params type for SVCB/HTTPS record: %T", params))
}

func MakeTLSA(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 4 {
		return nil, fmt.Errorf("MakeTLSA expects exactly 5 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.TLSA{Usage: mustbe.Uint8(args[0]), Selector: mustbe.Uint8(args[1]), MatchingType: mustbe.Uint8(args[2]), Certificate: mustbe.ToUpperRawString(args[3])}, nil
}

func MakeTXT(origin string, _ map[string]string, isEnabled nrc.Flags, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeTXT expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.TXT{Txt: mustbe.Txts(args[0])}, nil
}

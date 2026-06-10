package models

// Make*() functions for built-in types.
// TODO(tlim): Autogenerate this.  use mustbe.* while we're at it.

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	svcbv2 "codeberg.org/miekg/dns/svcb"

	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/privatetypes"
	_ "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
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

func MakeA(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeA expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	target := args[0]
	return dnsrdatav2.A{Addr: mustbe.IPv4(target)}, nil
}

func MakeALIAS(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeALIAS expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return privatetypesrdata.ALIAS{Target: mustbe.TargetHost(origin, args[0])}, nil
}
func MakeAAAA(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeAAAA expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.AAAA{Addr: mustbe.IPv6(args[0])}, nil
}

func MakeCAA(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 3 {
		return nil, fmt.Errorf("MakeCAA expects exactly 3 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.CAA{Flag: mustbe.Uint8(args[0]), Tag: mustbe.RawString(args[1]), Value: mustbe.RawString(args[2])}, nil
}
func MakeCNAME(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeCNAME expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.CNAME{Target: mustbe.TargetHost(origin, args[0])}, nil
}

// func MakeCFWORKERROUTE(origin string, when, then string) (dnsv2.RDATA, error) {
//	mustbe.ValidArgs(args)
// 	return privatetypesrdata.CFWORKERROUTE{When: mustbe.RawString(when), Then: mustbe.RawString(then)}, nil
// }

func MakeDHCID(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeDHCID expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.DHCID{Digest: mustbe.RawString(args[0])}, nil
}
func MakeDNAME(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeDNAME expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.DNAME{Target: mustbe.TargetHost(origin, args[0])}, nil
}
func MakeDNSKEY(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
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

func MakeDS(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 4 {
		return nil, fmt.Errorf("MakeDS expects exactly 4 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.DS{KeyTag: mustbe.Uint16(args[0]), Algorithm: mustbe.Uint8(args[1]), DigestType: mustbe.Uint8(args[2]), Digest: mustbe.RawString(args[3])}, nil
}

func MakeHTTPS(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 3 {
		return nil, fmt.Errorf("MakeHTTPS expects exactly 3 arguments, got %d: %+v", len(args), args)
	}
	return MakeSVCB(origin, nil, args[0], args[1], args[2])
}

func MakeLOC(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 7 {
		return nil, fmt.Errorf("MakeLOC expects exactly 7 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.LOC{
		Version: mustbe.Uint8(args[0]), Size: mustbe.Uint8(args[1]),
		HorizPre: mustbe.Uint8(args[2]), VertPre: mustbe.Uint8(args[3]),
		Latitude: mustbe.Uint32(args[4]), Longitude: mustbe.Uint32(args[5]),
		Altitude: mustbe.Uint32(args[6])}, nil
}

func MakeMIKROTIKFWD(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	return privatetypesrdata.MIKROTIKFWD{ForwardTo: mustbe.TargetHost(origin, args[0])}, nil
}
func MakeMIKROTIKNXDOMAIN(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	return privatetypesrdata.MIKROTIKNXDOMAIN{}, nil
}
func MakeMX(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 2 {
		return nil, fmt.Errorf("MakeMX expects exactly 2 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.MX{Preference: mustbe.Uint16(args[0]), Mx: mustbe.TargetHost(origin, args[1])}, nil
}

func MakeNS(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeNS expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.NS{Ns: mustbe.TargetHost(origin, args[0])}, nil
}
func MakeNAPTR(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 6 {
		return nil, fmt.Errorf("MakeNAPTR expects exactly 6 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.NAPTR{Order: mustbe.Uint16(args[0]), Preference: mustbe.Uint16(args[1]), Flags: mustbe.RawString(args[2]), Service: mustbe.RawString(args[3]), Regexp: mustbe.RawString(args[4]), Replacement: mustbe.TargetHost(origin, args[5])}, nil
}

func MakeOPENPGPKEY(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeOPENPGPKEY expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.OPENPGPKEY{PublicKey: mustbe.RawString(args[0])}, nil
}

func MakePORKBUNURLFWD(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	return privatetypesrdata.PORKBUNURLFWD{}, nil
}

func MakePTR(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakePTR expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.PTR{Ptr: mustbe.TargetHost(origin, args[0])}, nil
}

func MakeRP(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 2 {
		return nil, fmt.Errorf("MakeRP expects exactly 2 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.RP{Mbox: mustbe.TargetHost(origin, args[0]), Txt: mustbe.TargetHost(origin, args[1])}, nil
}

func MakeR53ALIAS(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 5 {
		return nil, fmt.Errorf("MakeR53ALIAS expects exactly 5 arguments, got %d: %+v", len(args), args)
	}
	return privatetypesrdata.R53ALIAS{
		AliasType: mustbe.RawString(args[0]),
		Target:    mustbe.TargetHost(origin, args[1]),
		// ZoneID:           mustbe.RawString(args[2]),
		// EvalTargetHealth: mustbe.RawString(args[3]),
		// FIXME(tlim): EvalTargetHealth is a boolean in our internal model but the R53ALIAS type expects a string. This is a hack to convert it to the expected format. We should probably change the R53ALIAS type to use a boolean for this field.
	}, nil
}

func MakeSMIMEA(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 4 {
		return nil, fmt.Errorf("MakeSMIMEA expects exactly 4 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.SMIMEA{Usage: mustbe.Uint8(args[0]), Selector: mustbe.Uint8(args[1]), MatchingType: mustbe.Uint8(args[2]), Certificate: mustbe.RawString(args[3])}, nil
}

func MakeSOA(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 7 {
		return nil, fmt.Errorf("MakeSOA expects exactly 9 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.SOA{
		Ns:      mustbe.TargetHost(origin, args[0]),
		Mbox:    mustbe.RawString(args[1]), // FIXME(tlim): Should be mustbe.SoaMbox()
		Serial:  mustbe.Uint32(args[2]),
		Refresh: mustbe.Uint32(args[3]),
		Retry:   mustbe.Uint32(args[4]),
		Expire:  mustbe.Uint32(args[5]),
		Minttl:  mustbe.Uint32(args[6]),
	}, nil
}

func MakeSRV(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 4 {
		return nil, fmt.Errorf("MakeSRV expects exactly 4 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.SRV{Priority: mustbe.Uint16(args[0]), Weight: mustbe.Uint16(args[1]), Port: mustbe.Uint16(args[2]), Target: mustbe.TargetHost(origin, args[3])}, nil
}

func MakeSSHFP(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 3 {
		return nil, fmt.Errorf("MakeSSHFP expects exactly 3 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.SSHFP{Algorithm: mustbe.Uint8(args[0]), Type: mustbe.Uint8(args[1]), FingerPrint: mustbe.RawString(args[2])}, nil
}

func MakeSVCB(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
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
		return dnsrdatav2.SVCB{Priority: mustbe.Uint16(priority), Target: mustbe.TargetHost(origin, target)}, nil
	}

	switch v := params.(type) {
	case []dnsv1.SVCBKeyValue:
		pv2, err := convertSVCBv1v2(v) // This hasn't tested extensively.
		if err != nil {
			panic("BUG: Failed to convert SVCB parameters from v1 to v2: " + err.Error())
		}
		return dnsrdatav2.SVCB{Priority: mustbe.Uint16(priority), Target: mustbe.TargetHost(origin, target), Value: pv2}, nil
	case []svcbv2.Pair:
		return dnsrdatav2.SVCB{Priority: mustbe.Uint16(priority), Target: mustbe.TargetHost(origin, target), Value: v}, nil
	case string:
		// fmt.Printf("DEBUG MakeSVCB: Before conversion params=%q\n", v)
		// v = strings.ReplaceAll(" "+v+" ", ` ech=IGNORE `, ` ech=1000`)
		// v = strings.ReplaceAll(v, `  `, ` `) // Collapse 2 spaces into 1  (This may be unneeded but doesn't hurt)
		// v = strings.TrimSpace(v)
		// fmt.Printf("DEBUG MakeSVCB: After conversion params=%q\n", v)
		// ech=1000 is a special value that indicates "use the ech value from
		// the existing zone." This is not an RFC standard, just something we do
		// in DNSControl. There is a very small chance that someone will
		// actually have an ech value of "0000" but if that happens I will eat
		// my hat.

		line := fmt.Sprintf("%d %s %s", mustbe.Uint16(priority), mustbe.TargetHost(origin, target), v)
		// fmt.Printf("DEBUG MakeSVCB: Creating RDATA with line=%q\n", line)
		return dnsv2.NewData(dnsv2.TypeHTTPS, line)
		// NB(tlim): It's an abomination to construct this string just to parse it but dnsv2 doesn't expose the parser in a way to do a partial line.
	}

	panic(fmt.Sprintf("BUG: Invalid params type for SVCB/HTTPS record: %T", params))
}

func MakeTLSA(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 4 {
		return nil, fmt.Errorf("MakeTLSA expects exactly 5 arguments, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.TLSA{Usage: mustbe.Uint8(args[0]), Selector: mustbe.Uint8(args[1]), MatchingType: mustbe.Uint8(args[2]), Certificate: mustbe.RawString(args[3])}, nil
}

func MakeTXT(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
	mustbe.ValidArgs(args)
	if len(args) != 1 {
		return nil, fmt.Errorf("MakeTXT expects exactly 1 argument, got %d: %+v", len(args), args)
	}
	return dnsrdatav2.TXT{Txt: mustbe.Txts(args[0])}, nil
}

// func MakeURL(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
//	mustbe.ValidArgs(args)
// 	if len(args) != 1 {
// 		return nil, fmt.Errorf("MakeURL expects exactly 1 argument, got %d: %+v", len(args), args)
// 	}
// 	return privatetypesrdata.URL{Location: mustbe.RawString(args[0])}, nil
// }

// func MakeURL301(origin string, _ map[string]string, args ...any) (dnsv2.RDATA, error) {
//	mustbe.ValidArgs(args)
// 	if len(args) != 1 {
// 		return nil, fmt.Errorf("MakeURL301 expects exactly 1 argument, got %d: %+v", len(args), args)
// 	}
// 	return privatetypesrdata.URL{Location: mustbe.RawString(args[0])}, nil
// }

package models

import (
	"fmt"
	"runtime/debug"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
)

// SetRDATA is a setter for RecordConfig.rdata.
func (rc *RecordConfig) SetRDATA(rd dnsv2.RDATA) {
	rc.rdata = assureNotPointerRDATA(rd)
	rc.ValidateRDATA()
	rc.ComparableV3 = ""
}

// GetRDATA is a getter for RecordConfig.rdata.
func (rc *RecordConfig) GetRDATA() (rd dnsv2.RDATA) {
	return rc.rdata
}

// ClearRDATA sets rc.rdata to nil. This is a workaround and will eventually be eliminated.
func (rc *RecordConfig) ClearRDATA() {
	rc.rdata = nil
	rc.ComparableV3 = ""
}

// ValidateRDATA is used to verify that .rdata didn't accidentally get set to
// rdata (instead of *rdata).  This shouldn't be needed, but it catches coding
// mistakes.  Eventually this may become a no-op.
func (rc *RecordConfig) ValidateRDATA() {

	if rc.GetRDATA() == nil {
		return
	}

	tn := fmt.Sprintf("%T", rc.GetRDATA())
	if strings.HasPrefix(tn, "rdata.") {
		return
	}
	if strings.HasPrefix(tn, "privatetypesrdata.") {
		return
	}

	l := fmt.Sprintf("\nDEBUG: ValidateRDATA: %s\n", tn)
	fmt.Println(l)
	fmt.Println(string(debug.Stack()))
	panic(l)
}

func MyNewData(typeNum uint16, contents string, origin string) (dnsv2.RDATA, error) {
	rd, err := dnsv2.NewData(typeNum, contents, origin+".")
	if err != nil {
		return nil, err
	}

	rd2 := assureNotPointerRDATA(rd)

	// TODO(tlim): This duplicates code in the MakeTYPE() functions, but
	// sadly those functions aren't called by dnsv2.NewData(). It is
	// unclear what would be better. Maybe privatetypes.RegisterMaker() can
	// also register a function that does this cleanup, and this
	// function would call privatetypes.PostParseCleanup(rd)?

	switch v := rd2.(type) {

	case dnsrdatav2.DS:
		v.Digest = strings.ToUpper(v.Digest)
		rd2 = v

	case dnsrdatav2.SSHFP:
		v.FingerPrint = strings.ToUpper(v.FingerPrint)
		rd2 = v

	case dnsrdatav2.TLSA:
		v.Certificate = strings.ToUpper(v.Certificate)
		rd2 = v

	case dnsrdatav2.TXT:
		// DNSControl stores TXT data as a single string (see models/t_txt.go); the
		// provider is responsible for splitting it into 255-octet segments on the
		// wire. The presentation-format parser, however, yields one Txt element per
		// segment, so a >255-octet TXT round-trips as multiple strings and its
		// RDATA.String() (used for ComparableV3) would differ from the same value
		// built via MakeTXT, causing a spurious diff. Rejoin into a single string.
		if len(v.Txt) > 1 {
			v.Txt = []string{strings.Join(v.Txt, "")}
			rd2 = v
		}

	}

	return rd2, nil
}

func assureNotPointerRDATA(rd dnsv2.RDATA) dnsv2.RDATA {

	//        Good: `rdata.A` or `privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	//         Bad: `*rdata.A` or `*privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	//  Really Bad: `**rdata.A` or `**privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	tn := fmt.Sprintf("%T", rd)
	if tn[0] != '*' {
		return rd
	}

	panic(fmt.Sprintf("########################## BROKEN %T", rd))
}

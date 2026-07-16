package models

import (
	"fmt"
	"os"
	"reflect"
	"runtime/debug"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"

	_ "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes"
	_ "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
)

// SetRDATA is a setter for RecordConfig.rdata.
func (rc *RecordConfig) SetRDATA(rd dnsv2.RDATA) {
	if txt, ok := rd.(dnsrdatav2.TXT); ok {
		txt.Txt = TXTSegmented(txt)
		rd = txt
	}
	rd = normalizeRDATA(rd)
	rc.rdata = rd
	rc.validateRDATA()
	rc.RegenerateComparableV3()
	if err := rc.copyRDtoLegacyFields(); err != nil {
		panic(err) // Should not happen.
	}
}

// GetRDATA is a getter for RecordConfig.rdata.
func (rc *RecordConfig) GetRDATA() (rd dnsv2.RDATA) {
	if rd, ok := rc.rdata.(dnsrdatav2.TXT); ok {
		if !txtProperlySegmented(rd.Txt) {
			fmt.Fprintf(os.Stderr, "WARNING: GetRDATA: TXT record not properly segmented. Someone is not using SetRDATA? txt=%+v\n", rd.Txt)
		}
	}
	return rc.rdata
}

// ClearRDATA sets rc.rdata to nil. This is a workaround and will eventually be eliminated.
func (rc *RecordConfig) ClearRDATA() {
	rc.rdata = nil
	rc.ComparableV3 = ""
}

func MyNewData(typeNum uint16, contents string, origin string) (dnsv2.RDATA, error) {
	rd2, err := dnsv2.NewData(typeNum, contents, origin+".")
	if err != nil {
		return nil, err
	}
	return normalizeRDATA(rd2), nil
}

// validateRDATA is used to verify that .rdata didn't accidentally get set to
// rdata (instead of *rdata).  This shouldn't be needed, but it catches coding
// mistakes.  Eventually this may become a no-op.
func (rc *RecordConfig) validateRDATA() {

	rd := rc.GetRDATA()
	if rd == nil {
		return
	}

	//        Good: `rdata.A` or `privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	//         Bad: `*rdata.A` or `*privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	//  Really Bad: `**rdata.A` or `**privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	if reflect.TypeOf(rd).Kind() != reflect.Pointer {
		return
	}
	// The common path uses reflection for speed. The old code looked like:
	// if ts := fmt.Sprintf("%T", rd); ts[0] != '*' {}

	// On the other hand, we use %T for the error path, which is rarely taken and can be slow.
	l := fmt.Sprintf("\nERROR: validateRDATA: typeNum=%d type=%q type=%T", rc.TypeNum, rc.Type, "%T", rd)
	fmt.Println(l)
	fmt.Println(string(debug.Stack()))
	panic(l)
}

func normalizeRDATA(rd2 dnsv2.RDATA) dnsv2.RDATA {
	// TODO(tlim): This duplicates code in the MakeTYPE() functions, but
	// sadly those functions aren't called by dnsv2.NewData().
	// Fixing this would be difficult since we can't add methods to the
	// dnsv2.RDATA interface.  We could use interfaces that only get called when they exist.

	switch v := rd2.(type) {

	case dnsrdatav2.DS:
		// Uppercase to make comparisons case-insensitive.
		v.Digest = strings.ToUpper(v.Digest)
		return v

	case dnsrdatav2.SSHFP:
		// Uppercase to make comparisons case-insensitive.
		v.FingerPrint = strings.ToUpper(v.FingerPrint)
		return v

	case dnsrdatav2.TLSA:
		// Uppercase to make comparisons case-insensitive.
		v.Certificate = strings.ToUpper(v.Certificate)
		return v

	case dnsrdatav2.TXT:
		// Store TXT data segments with all-but-the-last segment being exactly 255 octets.
		v.Txt = TXTSegmented(v)
		return v

	}

	return rd2
}

/*

FUTURE():

Add this interface. normalizeRDATA() will call the interface (if it exists for
the RDATA).  The type-specific code currently in normalizeRDATA will move to
files such as t_ds.go, t_sshfp.go, t_tlsa.go, t_txt.go.

type Normalizer interface {
	Normalize(*RecordConfig)
}

*/

/*

TODO():

Add this interface. Update pkg/normalize/validate.go to call the interface (if it exists for the RDATA).  The type-specific code currently
in pkg/normalize/validate.go files such as t_ds.go, t_sshfp.go, t_tlsa.go, t_txt.go (all in `models/`).
The functions rejectifTargetEqualsLabel and rejectifInvalidR53Weight providers/route53/auditrecords.go will move to models/t_r53alias.go

type Validater interface {
	Validate(*RecordConfig)
}

*/

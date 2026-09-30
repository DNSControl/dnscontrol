package models

//go:generate go run github.com/DNSControl/dnscontrol/v5/build/astypegen
//go:generate go run github.com/DNSControl/dnscontrol/v5/build/normalizergen

import (
	"fmt"
	"os"
	"reflect"
	"runtime/debug"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	_ "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	_ "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
)

// SetRDATA is a setter for RecordConfig.rdata.
func (rc *RecordConfig) SetRDATA(rd dnsv2.RDATA) {
	rc.rdata = normalizeRDATA(rd)
	rc.validateRDATA()
	rc.generateComparableV3()
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

// ClearRDATA sets rc.rdata to nil. This is a workaround for
// integrationTest/helpers_integration_test.go and will eventually be
// eliminated. No new uses, please!
func (rc *RecordConfig) ClearRDATA() {
	rc.rdata = nil
	rc.ComparableV3 = ""
}

func myNewData(typeNum uint16, contents string, origin string) (dnsv2.RDATA, error) {
	switch typeNum {

	case dnsv2.TypeTXT:
		// NewData expects quotes around TXT contents.
		if len(contents) > 0 && (contents[0] != '"' && contents[len(contents)-1] != '"') {
			contents = `"` + contents + `"`
		}

	}

	rd2, err := dnsv2.NewData(typeNum, contents, origin+".")
	if err != nil {
		return nil, fmt.Errorf("NewData(%d, %q, %q) failed: %w", typeNum, contents, origin+".", err)
	}
	// We do not need to call normalizeRDATA here because every caller to
	// myNewData eventually passes the result to newRecordConfigHelper,
	// which calls SetRDATA, which calls normalizeRDATA.
	return rd2, nil
}

// validateRDATA is used to verify that .rdata didn't accidentally get set to
// rdata (instead of *rdata).  This shouldn't be needed, but it catches coding
// mistakes.  Eventually this may become a no-op.
func (rc *RecordConfig) validateRDATA() {

	rd := rc.GetRDATA()
	if rd == nil {
		return
	}

	//        Good: `rdata.A`   or `privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	//         Bad: `*rdata.A`  or `*privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	//  Really Bad: `**rdata.A` or `**privatetypesrdata.CLOUDFLARE_WORKER_ROUTER`
	if reflect.TypeOf(rd).Kind() != reflect.Pointer {
		return
	}
	// The common path uses reflection for speed. The old code looked like:
	// if ts := fmt.Sprintf("%T", rd); ts[0] != '*' {}

	// On the other hand, we use %T for the error path, which is rarely taken and can be slow.
	l := fmt.Sprintf("\nERROR: validateRDATA: typeNum=%d type=%q type=%T", rc.TypeNum, rc.Type, rd)
	fmt.Println(l)
	fmt.Println(string(debug.Stack()))
	panic(l)
}

/*

TODO():

Add this interface. Update pkg/normalize/validate.go to call the interface (if it exists for the RDATA).  The type-specific code currently
in pkg/normalize/validate.go files such as t_ds.go, t_sshfp.go, t_tlsa.go, t_txt.go (all in `models/`).
The functions rejectifTargetEqualsLabel and rejectifInvalidR53Weight providers/route53/auditrecords.go will move to models/t_r53alias.go

type Validater interface {
	Validate(*RecordConfig)
}

*/

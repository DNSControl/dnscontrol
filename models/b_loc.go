package models

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

func init() {
	RegisterBuilder("LOC", BuilderLOC)
}

func BuilderLOC(dc *DomainConfig, ttl uint32, args []any) (Records, error) {

	if len(args) != 7 && len(args) != 12 {
		return nil, fmt.Errorf("LOC should have 7 or 12 parameters")
	}

	// if there are 7 args, this was preprocessed already.  Return a dnsvrdata2.LOC{} with those fields.
	if len(args) == 7 {

		rec, err := dc.NewRecordConfig(
			mustbe.RawString(args[0]),
			ttl,
			dnsv2.TypeLOC,
			args[1:],
			nil,
		)
		if err != nil {
			return nil,
				fmt.Errorf(
					"record error in BuilderLOC at [LOC(%s)]: %w",
					txtutil.ZoneifyManyAny(args), err)
		}
		return Records{rec}, nil
	}

	// if there are 12 args, pass them through the compiler.  Return a dnsvrdata2.LOC{} with the 7 fields.

	rc := &RecordConfig{
		Type:    "LOC",
		TypeNum: dnsv2.TypeLOC,
		TTL:     ttl,
	}
	name, _ := dc.LabelFromDnsconfigjs(args[0].(string))
	rc.Name = name
	rc.calculateLOCFields(
		mustbe.Uint8(args[1]),
		mustbe.Uint8(args[2]),
		mustbe.Float32(args[3]),
		mustbe.RawString(args[4]),
		mustbe.Uint8(args[5]),
		mustbe.Uint8(args[6]),
		mustbe.Float32(args[7]),
		mustbe.RawString(args[8]),
		mustbe.Float32(args[9]),
		mustbe.Float32(args[10]),
		mustbe.Float32(args[11]),
		mustbe.Float32(args[12]),
	)

	// if err != nil {
	// 	return nil,
	// 		fmt.Errorf(
	// 			"record error in BuilderLOC at [LOC(%s)]: %w",
	// 			txtutil.ZoneifyManyAny(args), err)
	// }

	rc.Type = "LOC"
	rc.TypeNum = dnsv2.TypeLOC
	backfill(rc)

	return Records{rc}, nil

}

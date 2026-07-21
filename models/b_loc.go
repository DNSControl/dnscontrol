package models

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

func init() {
	RegisterBuilder("LOC", BuilderLOC)
}

func BuilderLOC(dc *DomainConfig, ttl uint32, args []any, subdomain string) (Records, error) {

	// args includes the label at args[0], so the parameter counts are +1:
	// 8 = label + 7 preprocessed LOC fields; 13 = label + 12 DMS parameters.
	if len(args) != 8 && len(args) != 13 {
		return nil, fmt.Errorf("LOC should have 7 or 12 parameters")
	}

	// if there are 7 params, this was preprocessed already.  Return a dnsvrdata2.LOC{} with those fields.
	if len(args) == 8 {

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
	name, _ := dc.LabelFromDnsconfigjs(args[0].(string), subdomain)
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
		mustbe.Float64(args[9]),
		mustbe.Float32(args[10]),
		mustbe.Float32(args[11]),
		mustbe.Float32(args[12]),
	)

	// Populate the V3 fields (.RDATA and .ComparableV3) from the computed LOC
	// fields, matching the 8-arg path (which goes through NewRecordConfig).
	rc.RecomputeV3Fields(dc.Name)

	return Records{rc}, nil

}

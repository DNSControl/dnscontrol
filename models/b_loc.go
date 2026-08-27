package models

import (
	"errors"
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
		return nil, errors.New("LOC should have 7 or 12 parameters")
	}

	label, err := dc.LabelFromDnsconfigjs(
		mustbe.RawString(args[0]),
		subdomain,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid label in BuilderLOC at [LOC(%s)]: %w", txtutil.ZoneifyManyAny(args), err)
	}

	rec, err := dc.NewRecordConfig(
		label,
		ttl,
		dnsv2.TypeLOC,
		args[1:]...,
	)
	if err != nil {
		return nil, fmt.Errorf("record error in BuilderLOC at [LOC(%s)]: %w", txtutil.ZoneifyManyAny(args), err)
	}

	return Records{rec}, nil

}

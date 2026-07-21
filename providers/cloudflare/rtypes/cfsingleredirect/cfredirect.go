package cfsingleredirect

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

func init() {
	models.RegisterBuilder("CF_REDIRECT", BuilderCFREDIRECT)
	models.RegisterBuilder("CF_TEMP_REDIRECT", BuilderCFTEMPREDIRECT)
}

func BuilderCFREDIRECT(dc *models.DomainConfig, ttl uint32, args []any, subdomain string) (models.Records, error) {
	return builderCFREDIRECThelper(dc, 301, args, subdomain)
}
func BuilderCFTEMPREDIRECT(dc *models.DomainConfig, ttl uint32, args []any, subdomain string) (models.Records, error) {
	return builderCFREDIRECThelper(dc, 302, args, subdomain)
}

func builderCFREDIRECThelper(dc *models.DomainConfig, code uint16, args []any, subdomain string) (models.Records, error) {
	// Convert old-style patterns to new-style rules:
	prWhen := mustbe.RawString(args[1])
	prThen := mustbe.RawString(args[2])
	srWhen, srThen, err := makeRuleFromPattern(prWhen, prThen)
	if err != nil {
		return nil, err
	}

	// Create a rule name:
	name := fmt.Sprintf("%03d,%s,%s", code, prWhen, prThen)

	rec, err := dc.NewRecordConfig(
		"@",
		1, // CF ignores the TTL. We always force it to 1.
		"CLOUDFLAREAPI_SINGLE_REDIRECT",
		name, code, srWhen, srThen,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"record error in GeneratorCFREDIRECT at [CLOUDFLAREAPI_SINGLE_REDIRECT(%s)]: %w",
				txtutil.ZoneifyManyAny(args), err)
	}
	return models.Records{rec}, nil
}

package fortigate

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

// AuditRecords performs basic validation and returns warnings for known limitations.
func AuditRecords(records models.Records) []error {
	var problems []error

	for _, rc := range records {
		switch rc.TypeNum {
		case dnsv2.TypeA, dnsv2.TypeAAAA,
			// Supported
			// case "PTR":
			// 	// FortiGate limitations: these record types are not fully supported.
			// 	problems = append(problems,
			// 		fmt.Errorf("record type %s is not supported by FortiGate provider (name: %s)", rc.Type, rc.GetLabelFQDN()))
			dnsv2.TypeCNAME:

		default:
			problems = append(problems,
				fmt.Errorf("record type %s is not supported by FortiGate provider (name: %s)", rc.Type, rc.GetLabelFQDN()))
		}

		//Handle CNAME Records limitations
		if rc.Type == "CNAME" && rc.GetLabel() == "@" {
			problems = append(problems,
				fmt.Errorf("CNAME at apex (@) is not allowed (name: %s)", rc.GetLabelFQDN()))
		}

		// //Handle NS Records limitations
		// if rc.Type == "NS" && rc.GetLabel() != "@" && rc.GetLabel() != "" {
		// 	problems = append(problems,
		// 		fmt.Errorf("NS records are only supported at the zone apex (@): %s", rc.GetLabelFQDN()))
		// }

		// //Handle MX Records limitations
		// if rc.Type == "MX" {

		// 	// MX only supported at zone apex
		// 	if rc.GetLabel() != "@" && rc.GetLabel() != "" {
		// 		problems = append(problems,
		// 			fmt.Errorf("MX records are only supported at the zone apex (@): %s", rc.GetLabelFQDN()))
		// 	}

		// 	// FortiGate does not accept "." as target (it's not a valid DNS name)
		// 	target := rc.Get|TargetField()
		// 	if target == "." {
		// 		problems = append(problems,
		// 			fmt.Errorf("FortiGate does not accept '.' as an MX target: %s", rc.GetLabelFQDN()))
		// 	}
		// }

		// Wildcard support
		if strings.Contains(rc.GetLabelFQDN(), "*") {
			problems = append(problems,
				fmt.Errorf("wildcard record %s is not supported by FortiGate", rc.GetLabelFQDN()))
		}
	}

	return problems
}

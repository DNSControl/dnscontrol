package normalize

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func checkAutoDNSSEC(dc *models.DomainConfig) (errs []error) {
	if strings.ToLower(dc.RegistrarName) == "none" {
		return
	}
	if dc.AutoDNSSEC == "on" {
		for providerName := range dc.DNSProviderNames {
			if dc.RegistrarName != providerName {
				errs = append(errs, Warning{fmt.Errorf("AutoDNSSEC is enabled, but DNS provider %s does not match registrar %s", providerName, dc.RegistrarName)})
			}
		}
	}
	return
}

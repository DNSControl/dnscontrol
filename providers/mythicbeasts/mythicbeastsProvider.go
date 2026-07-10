// Package mythicbeasts provides a provider for managing zones in Mythic Beasts.
//
// This package uses the Primary DNS API v2, as described in https://www.mythic-beasts.com/support/api/dnsv2
package mythicbeasts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnsrr"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	dnsv1 "github.com/miekg/dns"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// mythicBeastsDefaultNS lists the default nameservers, per https://www.mythic-beasts.com/support/domains/nameservers.
var mythicBeastsDefaultNS = []string{
	"ns1.mythic-beasts.com",
	"ns2.mythic-beasts.com",
}

// mythicBeastsProvider is the handle for this provider.
type mythicBeastsProvider struct {
	client *http.Client
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Cannot("Requires domain registered through Web UI"),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "MYTHICBEASTS"
	const providerMaintainer = "@tomfitzhenry"
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Mythic Beasts",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/mythicbeasts",
		PortalURL:   "https://www.mythic-beasts.com/customer/api-users", // TODO: Verify
		Fields: []providers.CredsField{
			{
				Key:      "keyID",
				Label:    "Key ID",
				Help:     "Your Mythic Beasts API key ID.",
				Required: true,
			},
			{
				Key:      "secret",
				Label:    "Secret",
				Help:     "The secret paired with the key ID.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

func newDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	if conf["keyID"] == "" {
		return nil, errors.New("missing Mythic Beasts auth keyID")
	}
	if conf["secret"] == "" {
		return nil, errors.New("missing Mythic Beasts auth secret")
	}
	// Use https://www.mythic-beasts.com/support/api/auth
	cfg := clientcredentials.Config{
		ClientID:     conf["keyID"],
		ClientSecret: conf["secret"],
		TokenURL:     "https://auth.mythic-beasts.com/login",
		Scopes:       []string{"client_credentials"},
		AuthStyle:    oauth2.AuthStyleInHeader,
	}
	return &mythicBeastsProvider{
		client: cfg.Client(context.Background()),
	}, nil
}

func (n *mythicBeastsProvider) httpRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, "https://api.mythic-beasts.com/dns/v2"+url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "text/dns")
	req.Header.Add("Accept", "text/dns")
	return n.client.Do(req)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (n *mythicBeastsProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	resp, err := n.httpRequest("GET", "/zones/"+domain+"/records", nil)
	if err != nil {
		return nil, err
	}
	if got, want := resp.StatusCode, 200; got != want {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("got HTTP %v, want %v: %v", got, want, string(body))
	}
	return zoneFileToRecords(resp.Body, domain)
}

func zoneFileToRecords(r io.Reader, origin string) (models.Records, error) {
	zp := dnsv1.NewZoneParser(r, origin, origin)
	var records []*models.RecordConfig
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		rec, err := dnsrr.RRtoRC(rr, origin)
		if err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}

	if err := zp.Err(); err != nil {
		return nil, fmt.Errorf("parsing zone for %v: %w", origin, err)
	}
	return records, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (n *mythicBeastsProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, actual models.Records) ([]*models.Correction, int, error) {
	result, err := diff2.ByZone(actual, dc, nil)
	if err != nil {
		return nil, 0, err
	}
	msgs, changes, actualChangeCount := result.Msgs, result.HasChanges, result.ActualChangeCount

	var corrections []*models.Correction
	if changes {
		corrections = append(corrections,
			&models.Correction{
				Msg: strings.Join(msgs, "\n"),
				F: func() error {
					var b strings.Builder
					for _, record := range result.DesiredPlus {
						switch rr := record.ToRR().(type) {
						case *dnsv1.SSHFP:
							// "Hex strings [for SSHFP] must be in lower-case", per Mythic Beasts API docs.
							// miekg's DNS outputs uppercase: https://github.com/miekg/dns/blob/48f38ebef989eedc6b57f1869ae849ccc8f5fe29/types.go#L988
							fmt.Fprintf(&b, "%s %d %d %s\n", rr.Header().String(), rr.Algorithm, rr.Type, strings.ToLower(rr.FingerPrint))
						default:
							fmt.Fprintf(&b, "%v\n", rr.String())
						}
					}

					resp, err := n.httpRequest("PUT", "/zones/"+dc.Name+"/records", strings.NewReader(b.String()))
					if err != nil {
						return err
					}
					if got, want := resp.StatusCode, 200; got != want {
						body, _ := io.ReadAll(resp.Body)
						return fmt.Errorf("got HTTP %v, want %v: %v", got, want, string(body))
					}
					return nil
				},
			})
	}

	return corrections, actualChangeCount, nil
}

// GetNameservers returns the nameservers for a domain.
func (n *mythicBeastsProvider) GetNameservers(domainName string) ([]*models.Nameserver, error) {
	return models.ToNameservers(mythicBeastsDefaultNS)
}

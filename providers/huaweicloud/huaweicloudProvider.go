package huaweicloud

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/region"
	dnssdk "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
	dnsRegion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/region"
)

// Support for Huawei Cloud DNS.
// API Documentation: https://www.huaweicloud.com/intl/en-us/product/dns.html

/*
Huaweicloud API DNS provider:

Info required in `creds.json`:
   - KeyId
   - SecretKey
   - Region

Record level metadata available:
   - hw_line (refer below Huawei Cloud DNS API documentation for available lines, default "default_view")
             (https://support.huaweicloud.com/intl/en-us/api-dns/en-us_topic_0085546214.html)
   - hw_weight (0-1000, default "1")
   - hw_rrset_key (default "")

*/

type huaweicloudProvider struct {
	client         *dnssdk.DnsClient
	domainByZoneID map[string]string
	zoneIDByDomain map[string]string
	region         *region.Region
}

const (
	metaWeight    = "hw_weight"
	metaLine      = "hw_line"
	metaKey       = "hw_rrset_key"
	defaultWeight = "1"
	defaultLine   = "default_view"
)

// Initialize initializes a fresh provider instance.
func (c *huaweicloudProvider) Initialize(m map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	auth, err := basic.NewCredentialsBuilder().
		WithAk(m["KeyId"]).
		WithSk(m["SecretKey"]).
		SafeBuild()
	if err != nil {
		return err
	}
	region, err := dnsRegion.SafeValueOf(m["Region"])
	if err != nil {
		return err
	}

	client, err := dnssdk.DnsClientBuilder().
		WithRegion(region).
		WithCredential(auth).
		SafeBuild()
	if err != nil {
		return err
	}

	*c = huaweicloudProvider{
		client: dnssdk.NewDnsClient(client),
		region: region,
	}

	return nil
}

var defaultNameServerNames = []string{
	// DNS server for regions in the Chinese mainland
	"ns1.huaweicloud-dns.com.",
	"ns1.huaweicloud-dns.cn.",
	// DNS server for countries or regions outside the Chinese mainland
	"ns1.huaweicloud-dns.net.",
	"ns1.huaweicloud-dns.org.",
}

func init() {
	providers.Register[*huaweicloudProvider]("HUAWEICLOUD", providers.Definition{
		FriendlyName: "Huawei Cloud DNS",
		DocsURL:      "https://docs.dnscontrol.org/provider/huaweicloud",
		PortalURL:    "https://console-intl.huaweicloud.com/iam/?locale=en-us#/iam/users",
		CredFields: []providers.CredsField{
			{
				Key:      "KeyId",
				Label:    "Access key ID",
				Help:     "Your Huawei Cloud Access Key ID (AK).",
				Required: true,
			},
			{
				Key:      "SecretKey",
				Label:    "Secret access key",
				Help:     "Your Huawei Cloud Secret Access Key (SK).",
				Secret:   true,
				Required: true,
			},
			{
				Key:      "Region",
				Label:    "Region",
				Help:     "The Huawei Cloud region the DNS API call is routed through (for example ap-southeast-1).",
				Required: true,
			},
		},
		Maintainer: "@huihuimoe",
		Features: providers.DocumentationNotes{
			// The default for unlisted capabilities is 'Cannot'.
			// See providers/capabilities.go for the entire list of capabilities.
			providers.CanAutoDNSSEC:          providers.Can(),
			providers.CanGetZones:            providers.Can(),
			providers.CanUseAlias:            providers.Cannot(),
			providers.CanUseCAA:              providers.Can(),
			providers.CanUseDS:               providers.Cannot(),
			providers.CanUseLOC:              providers.Cannot(),
			providers.CanUseNAPTR:            providers.Cannot(),
			providers.CanUsePTR:              providers.Cannot(),
			providers.CanUseSRV:              providers.Can(),
			providers.CanUseSSHFP:            providers.Cannot(),
			providers.CanUseTLSA:             providers.Cannot(),
			providers.CanUseHTTPS:            providers.Cannot(),
			providers.CanUseSVCB:             providers.Cannot(),
			providers.CanUseSOA:              providers.Cannot(),
			providers.DocCreateDomains:       providers.Can(),
			providers.DocDualHost:            providers.Can(),
			providers.DocOfficiallySupported: providers.Cannot(),
		},
	})
}

// huaweicloud has request limiting like above.
// "The throttling threshold has been reached: policy user over ratelimit,limit:100,time:1 minute".
func withRetry(f func() error) {
	const maxRetries = 23
	const sleepTime = 5 * time.Second
	var currentRetry int
	for {
		err := f()
		if err == nil {
			return
		}
		if strings.Contains(err.Error(), "over ratelimit") {
			currentRetry++
			if currentRetry >= maxRetries {
				return
			}
			printer.Printf("Huaweicloud rate limit exceeded. Waiting %s to retry.\n", sleepTime)
			time.Sleep(sleepTime)
		} else {
			return
		}
	}
}

// GetNameservers returns the nameservers for a domain.
func (c *huaweicloudProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	if err := c.getZones(); err != nil {
		return nil, err
	}

	payload := &model.ShowPublicZoneNameServerRequest{
		ZoneId: c.zoneIDByDomain[domain],
	}
	res, err := c.client.ShowPublicZoneNameServer(payload)
	if err != nil {
		return nil, err
	}
	nameservers := []string{}
	if res.Nameservers != nil {
		for _, record := range *res.Nameservers {
			if record.Hostname != nil {
				nameservers = append(nameservers, *record.Hostname)
			}
		}
	}
	if len(nameservers) != 0 {
		return models.ToNameserversStripTD(nameservers)
	}

	return models.ToNameserversStripTD(defaultNameServerNames)
}

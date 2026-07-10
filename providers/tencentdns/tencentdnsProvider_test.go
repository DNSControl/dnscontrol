package tencentdns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/stretchr/testify/assert"
	intldomain "github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/domain/v20180808"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	domain "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/domain/v20180808"
)

func TestNewTencentDNS(t *testing.T) {
	config := map[string]string{
		"secret_id":  "test-id",
		"secret_key": "test-key",
		"region":     "ap-guangzhou",
	}

	provider, err := newTencentDNS(config)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.client)
	assert.False(t, provider.client.useIntlDomainClient)
	assert.NotNil(t, provider.client.domainClient)
	assert.Nil(t, provider.client.intlDomainClient)
}

func TestNewTencentDNS_IntlSite(t *testing.T) {
	config := map[string]string{
		"secret_id":  "test-id",
		"secret_key": "test-key",
		"region":     "ap-guangzhou",
		"site":       "intl",
	}

	provider, err := newTencentDNS(config)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.client)
	assert.True(t, provider.client.useIntlDomainClient)
	assert.Nil(t, provider.client.domainClient)
	assert.NotNil(t, provider.client.intlDomainClient)
}

func TestNewTencentDNS_MissingCreds(t *testing.T) {
	config := map[string]string{
		"secret_id": "test-id",
		// "secret_key" is missing
	}

	provider, err := newTencentDNS(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestNewTencentDNS_UnsupportedSite(t *testing.T) {
	config := map[string]string{
		"secret_id":  "test-id",
		"secret_key": "test-key",
		"site":       "moon",
	}

	provider, err := newTencentDNS(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "unsupported tencent cloud site")
}

func TestSiteConfigForSite(t *testing.T) {
	tests := []struct {
		name                string
		site                string
		endpoint            string
		useIntlDomainClient bool
	}{
		{
			name: "default",
		},
		{
			name: "china",
			site: "cn",
		},
		{
			name:                "intl",
			site:                "intl",
			endpoint:            intlDNSPodEndpoint,
			useIntlDomainClient: true,
		},
		{
			name:                "international",
			site:                "international",
			endpoint:            intlDNSPodEndpoint,
			useIntlDomainClient: true,
		},
		{
			name:                "mixed case",
			site:                "InTl",
			endpoint:            intlDNSPodEndpoint,
			useIntlDomainClient: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			siteConfig, err := siteConfigForSite(tc.site)
			assert.NoError(t, err)
			assert.Equal(t, tc.endpoint, siteConfig.dnspodEndpoint)
			assert.Equal(t, tc.useIntlDomainClient, siteConfig.useIntlDomainClient)
		})
	}
}

func TestPrepDesiredRecordsRewritesLowTTL(t *testing.T) {
	dc, _ := models.NewDomainConfig("example.com")
	dc.Records = models.Records{
		{TTL: 0},
		{TTL: 300},
		{TTL: 600},
		{TTL: 3600},
	}

	prepDesiredRecords(dc, 600)

	assert.Equal(t, uint32(0), dc.Records[0].TTL)
	assert.Equal(t, uint32(600), dc.Records[1].TTL)
	assert.Equal(t, uint32(600), dc.Records[2].TTL)
	assert.Equal(t, uint32(3600), dc.Records[3].TTL)
}

func TestPrepDesiredRecordsAllowsPaidDomainTTL(t *testing.T) {
	dc, _ := models.NewDomainConfig("example.com")
	dc.Records = models.Records{
		{TTL: 300},
	}

	prepDesiredRecords(dc, 1)

	assert.Equal(t, uint32(300), dc.Records[0].TTL)
}

func TestMinTTLForGrade(t *testing.T) {
	packages := []*dnspod.PackageDetailItem{
		{
			DomainGrade: new("DP_Free"),
			MinTtl:      new(uint64(600)),
		},
		{
			DomainGrade: new("DP_Plus"),
			MinTtl:      new(uint64(1)),
		},
		{
			DomainGrade: new("DP_MissingTTL"),
		},
	}

	assert.Equal(t, uint32(600), minTTLForGrade("DP_Free", packages))
	assert.Equal(t, uint32(1), minTTLForGrade("DP_Plus", packages))
	assert.Equal(t, defaultTTL, minTTLForGrade("DP_MissingTTL", packages))
	assert.Equal(t, defaultTTL, minTTLForGrade("DP_Unknown", packages))
}

func TestCredsMetadata(t *testing.T) {
	meta, ok := providers.GetCredsMetadata("TENCENTDNS")
	assert.True(t, ok)
	assert.Equal(t, "Tencent Cloud DNS", meta.DisplayName)
	assert.True(t, meta.Kind.Has(providers.KindDNS))
	assert.True(t, meta.Kind.Has(providers.KindRegistrar))
	assert.Equal(t, "https://docs.dnscontrol.org/provider/tencentdns", meta.DocsURL)
	assert.Equal(t, "https://console.intl.cloud.tencent.com/cam/capi", meta.PortalURL)

	if assert.Len(t, meta.Fields, 4) {
		assert.Equal(t, "secret_id", meta.Fields[0].Key)
		assert.True(t, meta.Fields[0].Required)
		assert.True(t, meta.Fields[0].Secret)

		assert.Equal(t, "secret_key", meta.Fields[1].Key)
		assert.True(t, meta.Fields[1].Required)
		assert.True(t, meta.Fields[1].Secret)

		assert.Equal(t, "region", meta.Fields[2].Key)
		assert.Equal(t, "ap-guangzhou", meta.Fields[2].Default)

		assert.Equal(t, "site", meta.Fields[3].Key)
		assert.Equal(t, "cn", meta.Fields[3].Default)
		assert.Contains(t, meta.Fields[3].Help, "international APIs")
	}
}

func TestDomainBatchStatus(t *testing.T) {
	details := []*domain.DomainBatchDetailSet{
		{
			Domain: new("example.com"),
			Status: new("failed"),
			Reason: new("invalid dns"),
		},
	}

	status, reason, found := domainBatchStatus(details, "EXAMPLE.COM")

	assert.True(t, found)
	assert.Equal(t, "failed", status)
	assert.Equal(t, "invalid dns", reason)
}

func TestDomainBatchStatusNotFound(t *testing.T) {
	status, reason, found := domainBatchStatus(nil, "example.com")

	assert.False(t, found)
	assert.Empty(t, status)
	assert.Empty(t, reason)
}

func TestIntlDomainBatchStatus(t *testing.T) {
	details := []*intldomain.BatchDomainBuyDetails{
		{
			Domain: new("example.com"),
			Status: new("FAILURE"),
			Reason: new("invalid dns"),
		},
	}

	status, reason, found := intlDomainBatchStatus(details, "EXAMPLE.COM")

	assert.True(t, found)
	assert.Equal(t, "FAILURE", status)
	assert.Equal(t, "invalid dns", reason)
}

func TestIntlDomainBatchStatusUsesReasonZh(t *testing.T) {
	details := []*intldomain.BatchDomainBuyDetails{
		{
			Domain:   new("example.com"),
			Status:   new("FAILURE"),
			ReasonZh: new("localized dns error"),
		},
	}

	status, reason, found := intlDomainBatchStatus(details, "example.com")

	assert.True(t, found)
	assert.Equal(t, "FAILURE", status)
	assert.Equal(t, "localized dns error", reason)
}

func TestIntlDomainBatchStatusNotFound(t *testing.T) {
	status, reason, found := intlDomainBatchStatus(nil, "example.com")

	assert.False(t, found)
	assert.Empty(t, status)
	assert.Empty(t, reason)
}

func TestNormalizeNameserverSet(t *testing.T) {
	got := normalizeNameserverSet([]string{
		"NANCY.NS.CLOUDFLARE.COM.",
		"rudy.ns.cloudflare.com",
	})

	assert.Equal(t, []string{"nancy.ns.cloudflare.com", "rudy.ns.cloudflare.com"}, got)
}

package tencentdns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
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
	dc := models.MustNewDomainConfig("example.com")
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
	dc := models.MustNewDomainConfig("example.com")
	dc.Records = models.Records{
		{TTL: 300},
	}

	prepDesiredRecords(dc, 1)

	assert.Equal(t, uint32(300), dc.Records[0].TTL)
}

func TestRecordLineComparableMatchesLineNameAndID(t *testing.T) {
	domain := "example.com"
	existingDefault := makeLineRecord(domain, "1.2.3.4", map[string]string{
		metaRecordLine:   defaultRecordLine,
		metaRecordLineID: "0",
	})
	existingTelecom := makeLineRecord(domain, "1.2.3.4", map[string]string{
		metaRecordLine:   "电信",
		metaRecordLineID: "10=1",
	})
	compare := recordMetadataComparable(models.Records{existingDefault, existingTelecom})

	desiredDefault := makeLineRecord(domain, "1.2.3.4", nil)
	desiredTelecomByName := makeLineRecord(domain, "1.2.3.4", map[string]string{
		metaRecordLine: "电信",
	})
	desiredTelecomByID := makeLineRecord(domain, "1.2.3.4", map[string]string{
		metaRecordLineID: "10=1",
	})

	assert.Equal(t, compare(existingDefault), compare(desiredDefault))
	assert.Equal(t, compare(existingTelecom), compare(desiredTelecomByName))
	assert.Equal(t, compare(existingTelecom), compare(desiredTelecomByID))
}

func TestRecordMetadataComparableDoesNotGuessAmbiguousLineID(t *testing.T) {
	domain := "example.com"
	first := makeLineRecord(domain, "192.0.2.1", map[string]string{
		metaRecordLine:   "custom",
		metaRecordLineID: "10=1",
	})
	second := makeLineRecord(domain, "192.0.2.1", map[string]string{
		metaRecordLine:   "custom",
		metaRecordLineID: "10=2",
	})
	desiredByName := makeLineRecord(domain, "192.0.2.1", map[string]string{
		metaRecordLine: "custom",
	})
	compare := recordMetadataComparable(models.Records{first, second})

	assert.NotEqual(t, compare(first), compare(desiredByName))
	assert.NotEqual(t, compare(second), compare(desiredByName))
}

func TestRecordMetadataComparableKeepsDefaultLineMapping(t *testing.T) {
	domain := "example.com"
	existingDefault := makeLineRecord(domain, "192.0.2.1", map[string]string{
		metaRecordLine: defaultRecordLine,
	})
	conflictingDefault := makeLineRecord(domain, "192.0.2.1", map[string]string{
		metaRecordLine:   defaultRecordLine,
		metaRecordLineID: "unexpected",
	})
	desiredDefault := makeLineRecord(domain, "192.0.2.1", nil)
	compare := recordMetadataComparable(models.Records{existingDefault, conflictingDefault})

	assert.Equal(t, compare(existingDefault), compare(desiredDefault))
	assert.NotEqual(t, compare(conflictingDefault), compare(desiredDefault))
}

func TestRecordLineParticipatesInDiff(t *testing.T) {
	domain := "example.com"
	existing := models.Records{
		makeLineRecord(domain, "1.2.3.4", map[string]string{
			metaRecordLine:   defaultRecordLine,
			metaRecordLineID: "0",
		}),
		makeLineRecord(domain, "1.2.3.4", map[string]string{
			metaRecordLine:   "电信",
			metaRecordLineID: "10=1",
		}),
	}
	dc := models.MustNewDomainConfig(domain)
	dc.Records = models.Records{
		makeLineRecord(domain, "1.2.3.4", nil),
		makeLineRecord(domain, "1.2.3.4", map[string]string{
			metaRecordLineID: "10=1",
		}),
	}

	changes, count, err := diff2.ByRecord(existing, dc, recordMetadataComparable(existing))

	assert.NoError(t, err)
	assert.Zero(t, count)
	assert.Empty(t, changes)
}

func TestRecordLineDiffKeepsDefaultDomesticAndForeignRecords(t *testing.T) {
	domain := "example.com"
	existing := models.Records{
		makeLineRecord(domain, "192.0.2.1", map[string]string{
			metaRecordLine:   defaultRecordLine,
			metaRecordLineID: "0",
		}),
		makeLineRecord(domain, "192.0.2.1", map[string]string{
			metaRecordLine:   "境内",
			metaRecordLineID: "3=0",
		}),
		makeLineRecord(domain, "192.0.2.2", map[string]string{
			metaRecordLine:   "境外",
			metaRecordLineID: "3=1",
		}),
	}
	dc := models.MustNewDomainConfig(domain)
	dc.Records = models.Records{
		makeLineRecord(domain, "192.0.2.1", nil),
		makeLineRecord(domain, "192.0.2.1", map[string]string{metaRecordLine: "境内"}),
		makeLineRecord(domain, "192.0.2.2", map[string]string{metaRecordLine: "境外"}),
	}

	changes, count, err := diff2.ByRecord(existing, dc, recordMetadataComparable(existing))

	assert.NoError(t, err)
	assert.Zero(t, count)
	assert.Empty(t, changes)
}

func TestRecordLineDiffDeletesOnlyExtraLine(t *testing.T) {
	domain := "example.com"
	existingDefault := makeLineRecord(domain, "1.2.3.4", map[string]string{
		metaRecordLine:   defaultRecordLine,
		metaRecordLineID: "0",
	})
	existingTelecom := makeLineRecord(domain, "1.2.3.4", map[string]string{
		metaRecordLine:   "电信",
		metaRecordLineID: "10=1",
	})
	existing := models.Records{existingDefault, existingTelecom}
	dc := models.MustNewDomainConfig(domain)
	dc.Records = models.Records{makeLineRecord(domain, "1.2.3.4", nil)}

	changes, count, err := diff2.ByRecord(existing, dc, recordMetadataComparable(existing))

	if assert.NoError(t, err) && assert.Len(t, changes, 1) {
		assert.Equal(t, diff2.DELETE, changes[0].Type)
		assert.Same(t, existingTelecom, changes[0].Old[0])
	}
	assert.Equal(t, 1, count)
}

func TestRecordWeightParticipatesInDiff(t *testing.T) {
	domain := "example.com"
	existing := models.Records{makeLineRecord(domain, "1.2.3.4", map[string]string{
		metaRecordWeight: "80",
	})}
	dc := models.MustNewDomainConfig(domain)
	dc.Records = models.Records{makeLineRecord(domain, "1.2.3.4", map[string]string{
		metaRecordWeight: "20",
	})}

	changes, count, err := diff2.ByRecord(existing, dc, recordMetadataComparable(existing))

	if assert.NoError(t, err) && assert.Len(t, changes, 1) {
		assert.Equal(t, diff2.CHANGE, changes[0].Type)
	}
	assert.Equal(t, 1, count)
}

func TestRecordWeightComparisonNormalizesValues(t *testing.T) {
	domain := "example.com"
	existing := models.Records{makeLineRecord(domain, "1.2.3.4", map[string]string{
		metaRecordWeight: "80",
	})}
	compare := recordMetadataComparable(existing)

	desiredWithLeadingZero := makeLineRecord(domain, "1.2.3.4", map[string]string{
		metaRecordWeight: "080",
	})
	desiredDisabled := makeLineRecord(domain, "1.2.3.4", map[string]string{
		metaRecordWeight: "0",
	})
	desiredUnset := makeLineRecord(domain, "1.2.3.4", nil)

	assert.Equal(t, compare(existing[0]), compare(desiredWithLeadingZero))
	assert.Equal(t, compare(desiredUnset), compare(desiredDisabled))
}

func makeLineRecord(domain, target string, metadata map[string]string) *models.RecordConfig {
	rc := &models.RecordConfig{
		Type:     "A",
		TTL:      600,
		Metadata: metadata,
	}
	rc.SetLabel("www", domain)
	rc.SetTarget(target)
	return rc
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

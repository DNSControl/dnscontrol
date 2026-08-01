package powerdns

import (
	"fmt"
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	svcbv2 "codeberg.org/miekg/dns/svcb"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/mittwald/go-powerdns/apis/zones"
	"github.com/stretchr/testify/assert"
)

func TestToRecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	record := zones.Record{
		Content: "simple",
	}
	recordConfig, err := toRecordConfig(dc, record, 120, "test", "TXT")

	assert.NoError(t, err)
	assert.Equal(t, "test.example.com", recordConfig.NameFQDN)
	assert.Equal(t, "\"simple\"", recordConfig.GetRDATA().String())
	assert.Equal(t, uint32(120), recordConfig.TTL)
	assert.Equal(t, "TXT", recordConfig.Type)

	largeContent := fmt.Sprintf("\"%s\" \"%s\"", strings.Repeat("A", 300), strings.Repeat("B", 300))
	largeRecord := zones.Record{
		Content: largeContent,
	}
	recordConfig, err = toRecordConfig(dc, largeRecord, 5, "large", "TXT")

	assert.NoError(t, err)
	assert.Equal(t, "large.example.com", recordConfig.NameFQDN)
	assert.Equal(t, `"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB" "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"`,
		recordConfig.GetRDATA().String())
	assert.Equal(t, uint32(5), recordConfig.TTL)
	assert.Equal(t, "TXT", recordConfig.Type)

	luaRecord := zones.Record{
		Content: "TXT \"return 'Hello, world!'\"",
	}
	recordConfig, err = toRecordConfig(dc, luaRecord, 3600, "script", "LUA")

	assert.NoError(t, err)
	assert.Equal(t, "script.example.com", recordConfig.NameFQDN)
	assert.Equal(t, "LUA", recordConfig.Type)
	assert.Equal(t, "TXT", recordConfig.LuaRType)
	assert.Equal(t, "return 'Hello, world!'", recordConfig.GetTargetTXTJoined())
	assert.Equal(t, "TXT \"return 'Hello, world!'\"", recordConfig.GetRDATA().String())
	assert.Equal(t, uint32(3600), recordConfig.TTL)

	autoHintRecord := zones.Record{
		Content: "1 . alpn=h3,h2 ipv4hint=auto ipv6hint=auto",
	}
	recordConfig, err = toRecordConfig(dc, autoHintRecord, 300, "auto", "HTTPS")

	assert.NoError(t, err)
	assert.Equal(t, "auto.example.com", recordConfig.NameFQDN)
	assert.Equal(t, "HTTPS", recordConfig.Type)
	assert.Equal(t, uint16(1), recordConfig.SvcPriority)
	assert.Equal(t, ".", recordConfig.GetTargetField())
	assert.Equal(t, "alpn=h3,h2 ipv4hint=auto ipv6hint=auto", recordConfig.SvcParams)
	assert.Equal(t, "1 . alpn=h3,h2 ipv4hint=auto ipv6hint=auto", powerDNSTargetCombined(recordConfig))
	assert.Equal(t, uint32(300), recordConfig.TTL)
}

func TestBuildRecordListSvcbAutoHints(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	recordConfig := dc.MustNewRecordConfig("auto", 300, dnsv2.TypeHTTPS, uint16(1), ".", []svcbv2.Pair{})
	recordConfig.SvcParams = "alpn=h3,h2 ipv4hint=auto ipv6hint=auto"

	records := buildRecordList(diff2.Change{New: models.Records{recordConfig}})

	assert.Len(t, records, 1)
	assert.Equal(t, "1 . alpn=h3,h2 ipv4hint=auto ipv6hint=auto", records[0].Content)
}

func TestBuildRecordListSvcbEchKeepsQuotes(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	recordConfig := dc.MustNewRecordConfig("@", 300, "SVCB", uint16(3), "example.com.", "alpn=h2,h3 port=999 ech=some+base64+encoded+value///")

	records := buildRecordList(diff2.Change{New: models.Records{recordConfig}})

	assert.Len(t, records, 1)
	assert.Equal(t, `3 example.com. alpn=h2,h3 port=999 ech="some+base64+encoded+value///"`, records[0].Content)
}

func TestParseText(t *testing.T) {
	// short TXT record
	short := parseTxt("\"simple\"")
	assert.Equal(t, []string{"simple"}, short)

	// TXT record with multiple parts
	multiple := parseTxt("\"simple\" \"simple2\"")
	assert.Equal(t, []string{"simple", "simple2"}, multiple)

	// long TXT record
	long := parseTxt(fmt.Sprintf("\"%s\"", strings.Repeat("A", 300)))
	assert.Equal(t, []string{strings.Repeat("A", 300)}, long)

	// multiple long TXT record
	multipleLong := parseTxt(fmt.Sprintf("\"%s\" \"%s\"", strings.Repeat("A", 300), strings.Repeat("B", 300)))
	assert.Equal(t, []string{strings.Repeat("A", 300), strings.Repeat("B", 300)}, multipleLong)
}

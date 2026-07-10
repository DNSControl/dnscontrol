package netnod

import (
	"fmt"
	"strings"
	"testing"

	netnodPrimaryDNS "github.com/netnod/netnod-primary-dns-client"
	"github.com/stretchr/testify/assert"
)

func TestToRecordConfig(t *testing.T) {
	record := netnodPrimaryDNS.Record{
		Content: "simple",
	}
	recordConfig, err := toRecordConfig("example.com", record, 120, "test", "TXT")

	assert.NoError(t, err)
	assert.Equal(t, "test.example.com", recordConfig.NameFQDN)
	assert.Equal(t, "\"simple\"", recordConfig.String())
	assert.Equal(t, uint32(120), recordConfig.TTL)
	assert.Equal(t, "TXT", recordConfig.Type)

	largeContent := fmt.Sprintf("\"%s\" \"%s\"", strings.Repeat("A", 300), strings.Repeat("B", 300))
	largeRecord := netnodPrimaryDNS.Record{
		Content: largeContent,
	}
	recordConfig, err = toRecordConfig("example.com", largeRecord, 5, "large", "TXT")

	assert.NoError(t, err)
	assert.Equal(t, "large.example.com", recordConfig.NameFQDN)
	assert.Equal(t, `"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB" "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"`,
		recordConfig.String())
	assert.Equal(t, uint32(5), recordConfig.TTL)
	assert.Equal(t, "TXT", recordConfig.Type)

	luaRecord := netnodPrimaryDNS.Record{
		Content: "TXT \"return 'Hello, world!'\"",
	}
	recordConfig, err = toRecordConfig("example.com", luaRecord, 3600, "script", "LUA")

	assert.NoError(t, err)
	assert.Equal(t, "script.example.com", recordConfig.NameFQDN)
	assert.Equal(t, "LUA", recordConfig.Type)
	assert.Equal(t, "TXT", recordConfig.LuaRType)
	assert.Equal(t, "return 'Hello, world!'", recordConfig.GetTargetTXTJoined())
	assert.Equal(t, "TXT \"return 'Hello, world!'\"", recordConfig.GetTargetCombined())
	assert.Equal(t, uint32(3600), recordConfig.TTL)
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

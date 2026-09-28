package realtimeregister

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/stretchr/testify/assert"
)

func TestRemoveEscapeChars(t *testing.T) {
	cleanedString := removeEscapeChars("\\\\\\\"")
	assert.Equal(t, "\\\"", cleanedString)
}

func TestAddEscapeChars(t *testing.T) {
	addedString := addEscapeChars("\\\"")
	assert.Equal(t, "\\\\\\\"", addedString)
}

func TestTxtRecordQuotes(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	testStrings := [...]string{"unquoted", "\"quoted\"", "embedded \"quotes\" in string"}

	for _, testString := range testStrings {
		rc := dc.MustNewRecordConfig("@", 0, dnsv2.TypeTXT, testString)
		record := toRecord(rc)

		assert.Equal(t, testString, record.Content)
	}
}

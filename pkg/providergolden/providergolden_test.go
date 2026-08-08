package providergolden

import (
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestConversionFilenames(t *testing.T) {
	domain := "example.com"
	tests := map[string]string{
		toRCInputFile("convert", domain):      "recorded_torc_input_convert_example.com.json",
		toRCOutputFile("convert", domain):     "expected_torc_output_convert_example.com.records",
		toNativeInputFile("convert", domain):  "recorded_tonative_input_convert_example.com.records",
		toNativeOutputFile("convert", domain): "expected_tonative_output_convert_example.com.json",
	}
	for got, want := range tests {
		if got != want {
			t.Errorf("filename = %q, want %q", got, want)
		}
	}
}

func TestParseIndexedRecordsGroupsRepeatedIndexes(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	groups, err := parseIndexedRecords(dc, strings.Join([]string{
		"1\twww 300 IN A 192.0.2.1",
		"1\twww 300 IN AAAA 2001:db8::1",
		"2\t@ 300 IN TXT \"hello\"",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0].Index != 1 || len(groups[0].Records) != 2 || groups[1].Index != 2 || len(groups[1].Records) != 1 {
		t.Fatalf("unexpected groups: %#v", groups)
	}
}

func TestParseIndexedRecordsRejectsOutOfOrderIndexes(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	_, err := parseIndexedRecords(dc, "2\twww 300 IN A 192.0.2.1\n1\tmail 300 IN A 192.0.2.2\n")
	if err == nil || !strings.Contains(err.Error(), "indexes must increase") {
		t.Fatalf("error = %v, want index-order error", err)
	}
}

func TestParseRecordPreservesMetadata(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc, err := parseRecord(dc, `www 300 IN A 192.0.2.1 ; key="value with spaces"`)
	if err != nil {
		t.Fatal(err)
	}
	if rc.Metadata["key"] != "value with spaces" {
		t.Fatalf("metadata = %#v", rc.Metadata)
	}
}

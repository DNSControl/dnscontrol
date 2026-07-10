package vultr

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/vultr/govultr/v2"
)

func TestConversion(t *testing.T) {
	dc, _ := models.NewDomainConfig("example.com")

	records := []govultr.DomainRecord{
		{
			Type: "A",
			Name: "",
			Data: "127.0.0.1",
			TTL:  300,
		},
		{
			Type: "CNAME",
			Name: "*",
			Data: "example.com",
			TTL:  300,
		},
		{
			Type:     "SRV",
			Name:     "_ssh_.tcp",
			Data:     "5 22 ssh.example.com",
			Priority: 5,
			TTL:      300,
		},
		{
			Type:     "MX",
			Name:     "",
			Data:     "mail.example.com",
			Priority: 0,
			TTL:      300,
		},
		{
			Type: "NS",
			Name: "",
			Data: "ns1.example.net",
			TTL:  300,
		},
		{
			Type: "TXT",
			Name: "test",
			Data: "\"testasd asda sdas dasd\"",
			TTL:  300,
		},
		{
			Type: "CAA",
			Name: "testasd",
			Data: "0 issue \"test.example.net\"",
			TTL:  300,
		},
	}

	for _, record := range records {
		rc, err := toRecordConfig(dc.Name, record)
		if err != nil {
			t.Error("Error converting Vultr record", record)
		}

		converted := toVultrRecord(rc, "0")

		if converted.Type != record.Type || converted.Name != record.Name || converted.Data != record.Data || (converted.Priority != record.Priority) || converted.TTL != record.TTL {
			t.Error("Vultr record conversion mismatch", record, rc, converted)
		}
	}
}

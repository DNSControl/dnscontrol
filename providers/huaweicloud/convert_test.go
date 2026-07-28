package huaweicloud

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)

func TestNativeToRecords(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	name, rtype, ttl := "www.example.com.", "A", int32(300)
	values := []string{"192.0.2.1", "192.0.2.2"}
	records, err := nativeToRecords(&model.ShowRecordSetByZoneResp{
		Name: &name, Type: &rtype, Ttl: &ttl, Records: &values,
	}, dc)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	for i, want := range values {
		if got := records[i].GetRDATA().String(); got != want {
			t.Errorf("record %d target = %q, want %q", i, got, want)
		}
	}
}

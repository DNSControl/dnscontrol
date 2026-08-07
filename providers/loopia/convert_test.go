package loopia

import (
	"reflect"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestRecordToNative_1(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	rc := dc.MustNewRecordConfig("foo", 3600, "A", "1.2.3.4")

	nst := reflect.TypeOf(recordToNative(rc)).Kind()
	if nst != reflect.TypeFor[paramStruct]().Kind() {
		t.Errorf("recordToNative produced unexpected type")
	}
}

func TestNativeToRecord_1(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	zrec := zRec{}
	zrec.Type = "A"
	zrec.TTL = 300
	zrec.Rdata = "1.2.3.4"
	zrec.Priority = 0
	zrec.RecordID = 0

	rc, err := nativeToRecord(zrec.SetZR(), dc, "www")

	if rc.Type != "A" {
		t.Errorf("nativeToRecord produced unexpected type")
	} else if rc.TTL != 300 {
		t.Errorf("nativeToRecord produced unexpected TTL")
	} else if rc.AsA().String() != "1.2.3.4" {
		t.Errorf("nativeToRecord produced unexpected Rdata")
	}

	if err != nil {
		t.Errorf("nativeToRecord error")
	}
}

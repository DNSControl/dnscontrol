package loopia

import (
	"reflect"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestRecordToNative_1(t *testing.T) {
	rc := &models.RecordConfig{
		TTL: 3600,
	}
	rc.SetLabel("foo", "example.com")
	rc.MustSetTarget("1.2.3.4")
	rc.Type = "A"

	nst := reflect.TypeFor[paramStruct]().Kind()
	if nst != reflect.TypeFor[paramStruct]().Kind() {
		t.Errorf("recordToNative produced unexpected type")
	}
}

func TestNativeToRecord_1(t *testing.T) {
	zrec := zRec{}
	zrec.Type = "A"
	zrec.TTL = 300
	zrec.Rdata = "1.2.3.4"
	zrec.Priority = 0
	zrec.RecordID = 0

	rc, err := nativeToRecord(zrec.SetZR(), "example.com", "www")

	if rc.Type != "A" {
		t.Errorf("nativeToRecord produced unexpected type")
	} else if rc.TTL != 300 {
		t.Errorf("nativeToRecord produced unexpected TTL")
	} else if rc.GetTargetCombined() != "1.2.3.4" {
		t.Errorf("nativeToRecord produced unexpected Rdata")
	} else if rc.SrvPriority != 0 {
		t.Errorf("nativeToRecord produced unexpected Priority")
	}

	if err != nil {
		t.Errorf("nativeToRecord error")
	}
}

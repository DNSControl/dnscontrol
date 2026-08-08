package gcloud

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	gdns "google.golang.org/api/dns/v1"
)

func TestNativeToRecordGolden(t *testing.T) {
	providergolden.CheckToRC(t, "nativeToRecord",
		func(dc *models.DomainConfig, native gdns.ResourceRecordSet) ([]*models.RecordConfig, error) {
			// GCLOUD returns every value of a label/rtype pair in one set.
			rcs := make([]*models.RecordConfig, 0, len(native.Rrdatas))
			for _, rdata := range native.Rrdatas {
				rc, err := nativeToRecord(&native, rdata, dc)
				if err != nil {
					return nil, err
				}
				rcs = append(rcs, rc)
			}
			return rcs, nil
		})
}

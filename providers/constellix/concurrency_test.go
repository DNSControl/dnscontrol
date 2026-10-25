package constellix

import (
	"fmt"
	"sync"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// TestConcurrentReads reads several zones at once, the way DNSControl gathers
// zones concurrently, while one of the reads is answered with a 429. Run it
// with -race.
func TestConcurrentReads(t *testing.T) {
	api := newFakeAPI(t)
	api.zones = nil
	for i := range 6 {
		api.zones = append(api.zones, map[string]any{"id": i, "name": fmt.Sprintf("z%d.example", i), "nameservers": testNameservers})
		api.records[fmt.Sprint(i)] = fmt.Sprintf(`[{"id": %d, "name": "www", "type": "A", "ttl": 300, "value": [{"value": "192.0.2.%d", "enabled": true}]}]`, 100+i, i)
	}
	api.rateLimitOnce = map[string]bool{"/domains/3/records": true}
	p := testProvider(t, api)

	var wg sync.WaitGroup
	errs := make([]error, len(api.zones))
	counts := make([]int, len(api.zones))
	for i := range api.zones {
		wg.Go(func() {
			name := fmt.Sprintf("z%d.example", i)
			if _, err := p.GetNameservers(name); err != nil {
				errs[i] = err
				return
			}
			recs, err := p.GetZoneRecords(models.MustNewDomainConfig(name))
			errs[i], counts[i] = err, len(recs)
		})
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil || counts[i] != 1 {
			t.Errorf("zone %d: %d records, error %v; want 1 record", i, counts[i], err)
		}
	}
	if len(api.rateLimitOnce) != 0 {
		t.Error("the 429 was never served")
	}
}

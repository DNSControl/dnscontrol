package mikrotik

import (
	"encoding/json"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

// Test adapters preserve existing constructor fixtures while exercising Initialize.
func newMikrotikProvider(cfg map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	p := new(mikrotikProvider)
	if err := p.Initialize(cfg, nil, nil); err != nil {
		return nil, err
	}
	return p, nil
}

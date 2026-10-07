package tencentdns

// Test adapters preserve existing constructor fixtures while exercising Initialize.
func newTencentDNS(config map[string]string) (*tencentdnsProvider, error) {
	p := new(tencentdnsProvider)
	if err := p.Initialize(config, nil, nil); err != nil {
		return nil, err
	}
	return p, nil
}

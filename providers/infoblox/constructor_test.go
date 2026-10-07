package infoblox

// Test adapters preserve existing constructor fixtures while exercising Initialize.
func newInfoblox(conf map[string]string) (*infobloxProvider, error) {
	p := new(infobloxProvider)
	if err := p.Initialize(conf, nil, nil); err != nil {
		return nil, err
	}
	return p, nil
}

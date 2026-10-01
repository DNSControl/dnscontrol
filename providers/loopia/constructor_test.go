package loopia

// Test adapters preserve existing constructor fixtures while exercising Initialize.
func NewClient(apiUser, apiPassword string, region string, modifyns bool, fetchns bool, debug bool) *APIClient {
	p := new(APIClient)
	p.initializeClient(apiUser, apiPassword, region, modifyns, fetchns, debug)
	return p
}

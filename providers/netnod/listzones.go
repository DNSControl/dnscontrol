package netnod

import "strings"

// ListZones returns all the zones in an account.
func (dsp *netnodProvider) ListZones() ([]string, error) {
	zones, err := dsp.client.ListZones()
	if err != nil {
		return nil, err
	}

	var result []string
	for _, zone := range zones {
		result = append(result, strings.TrimSuffix(zone.Name, "."))
	}
	return result, nil
}

package spaceship

// ListZones returns every domain in the Spaceship account.
func (c *spaceshipProvider) ListZones() ([]string, error) {
	list, err := c.getDomainList()
	if err != nil {
		return nil, err
	}
	zones := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		name := item.Name
		if name == "" {
			name = item.UnicodeName
		}
		if name != "" {
			zones = append(zones, name)
		}
	}
	return zones, nil
}

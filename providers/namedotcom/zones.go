package namedotcom

import "github.com/namedotcom/go/v4/namecom"

// ListZones returns all the zones in an account.
func (n *namedotcomProvider) ListZones() ([]string, error) {
	var names []string
	var page int32

	for {
		response, err := n.client.ListDomains(&namecom.ListDomainsRequest{Page: page})
		if err != nil {
			return nil, err
		}
		page = response.NextPage

		for _, j := range response.Domains {
			names = append(names, j.DomainName)
		}

		if page == 0 {
			break
		}
	}

	return names, nil
}

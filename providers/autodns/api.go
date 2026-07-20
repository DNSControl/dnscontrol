package autodns

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// ZoneListFilter describes a JSON list filter.
type ZoneListFilter struct {
	Key      string            `json:"key"`
	Value    string            `json:"value"`
	Operator string            `json:"operator"`
	Link     string            `json:"link,omitempty"`
	Filter   []*ZoneListFilter `json:"filters,omitempty"`
}

// ZoneListRequest describes a JSON zone list request.
type ZoneListRequest struct {
	Filter []*ZoneListFilter `json:"filters"`
	View   *ZoneListView     `json:"view,omitempty"`
}

// ZoneListView describes a JSON zone list reply.
type ZoneListView struct {
	Limit    int  `json:"limit"`
	Offset   int  `json:"offset"`
	Children bool `json:"children"`
}

var (
	// Default retry configuration.
	defaultRetryWait = 3 * time.Second
	defaultRetryMax  = 4
)

func (api *autoDNSProvider) request(method string, requestPath string, data any) ([]byte, error) {
	var retryCounter = 0

	client := &http.Client{}

	requestURL := api.baseURL
	requestURL.Path = api.baseURL.Path + requestPath

	request := &http.Request{
		URL:    &requestURL,
		Header: api.defaultHeaders,
		Method: method,
	}

	if data != nil {
		body, _ := json.Marshal(data)
		buffer := bytes.NewBuffer(body)
		request.Body = io.NopCloser(buffer)
	}

	for {
		response, err := client.Do(request)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()

		responseText, _ := io.ReadAll(response.Body)

		// FUTUREWORK: 202 starts a long-running task. Should we instead poll here until task is completed?
		if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusTooManyRequests {
			return nil, errors.New("Request to " + requestURL.Path + " failed: " + string(responseText))
		}

		if response.StatusCode == http.StatusOK {
			return responseText, nil
		}

		if retryCounter == defaultRetryMax { // the condition stops matching
			break // break out of the loop
		}

		retryCounter++

		sleepDuration := time.Duration(math.Pow(2, float64(retryCounter)) * float64(defaultRetryWait))

		time.Sleep(sleepDuration)
	}

	return nil, errors.New("Failed to fetch" + requestURL.Path + " after 4 retries")
}

func (api *autoDNSProvider) findZoneSystemNameServer(domain string) (*models.Nameserver, error) {
	request := &ZoneListRequest{}

	// When opted in, request child zones so master/admin users can locate
	// zones owned by sub-users (the optional "include subusers" UI toggle).
	if api.includeChildren {
		request.View = &ZoneListView{Limit: 1, Offset: 0, Children: true}
	}

	request.Filter = append(request.Filter, &ZoneListFilter{
		Key:      "name",
		Value:    domain,
		Operator: "EQUAL",
	})

	responseData, err := api.request("POST", "zone/_search", request)
	if err != nil {
		return nil, err
	}

	var responseObject JSONResponseDataZone
	if err := json.Unmarshal(responseData, &responseObject); err != nil {
		return nil, err
	}

	if len(responseObject.Data) != 1 {
		return nil, fmt.Errorf("Zone "+domain+" could not be found in AutoDNS: %w", os.ErrNotExist)
	}

	systemNameServer := &models.Nameserver{Name: responseObject.Data[0].SystemNameServer}

	return systemNameServer, nil
}

func (api *autoDNSProvider) createZone(domain string, zone *Zone) (*Zone, error) {
	responseData, err := api.request("POST", "zone", zone)
	if err != nil {
		return nil, err
	}

	var responseObject JSONResponseDataZone
	if err := json.Unmarshal(responseData, &responseObject); err != nil {
		return nil, err
	}

	if len(responseObject.Data) != 1 {
		return nil, errors.New("Zone " + domain + " not returned")
	}

	return responseObject.Data[0], nil
}

func (api *autoDNSProvider) getDomain(domain string) (*Domain, error) {
	responseData, err := api.request("GET", "domain/"+domain, nil)
	if err != nil {
		return nil, err
	}

	var responseObject JSONResponseDataDomain
	if err := json.Unmarshal(responseData, &responseObject); err != nil {
		return nil, err
	}

	if len(responseObject.Data) != 1 {
		return nil, fmt.Errorf("Domain "+domain+" could not be found in AutoDNS: %w", os.ErrNotExist)
	}

	return responseObject.Data[0], nil
}

func (api *autoDNSProvider) getZone(domain string) (*Zone, error) {
	systemNameServer, err := api.findZoneSystemNameServer(domain)
	if err != nil {
		return nil, err
	}

	// if resolving of a systemNameServer succeeds the system contains this zone
	responseData, err2 := api.request("GET", "zone/"+domain+"/"+systemNameServer.Name, nil)

	if err2 != nil {
		return nil, err2
	}

	var responseObject JSONResponseDataZone
	// make sure that the response is valid, the zone is in AutoDNS but we're not sure the returned data meets our expectation
	if err := json.Unmarshal(responseData, &responseObject); err != nil {
		return nil, err
	}

	return responseObject.Data[0], nil
}

func (api *autoDNSProvider) getZones() ([]string, error) {
	countZoneRequest := &ZoneListRequest{
		View: &ZoneListView{
			Limit:    0,
			Offset:   0,
			Children: api.includeChildren,
		},
	}

	rawCountZoneResponse, _ := api.request("POST", "zone/_search", countZoneRequest)

	var countZoneResponse JSONResponseDataZone
	if err := json.Unmarshal(rawCountZoneResponse, &countZoneResponse); err != nil {
		return nil, err
	}

	i := 0
	names := make([]string, 0, countZoneResponse.MetaData.ItemCount)

	// stop after a max of 99 iterations or if all zones are fetched
	for i < 99 && len(names) != countZoneResponse.MetaData.ItemCount {
		request := &ZoneListRequest{
			View: &ZoneListView{
				Limit:    100,
				Offset:   i * 100,
				Children: api.includeChildren,
			},
		}

		responseData, err := api.request("POST", "zone/_search", request)
		if err != nil {
			return nil, err
		}

		var responseObject JSONResponseDataZone
		if err := json.Unmarshal(responseData, &responseObject); err != nil {
			return nil, err
		}

		for _, zone := range responseObject.Data {
			names = append(names, zone.Origin)
		}

		i++
	}

	return names, nil
}

func (api *autoDNSProvider) updateZone(domain string, resourceRecords []*ResourceRecord, nameServers []*models.Nameserver, zoneTTL uint32) error {
	systemNameServer, err := api.findZoneSystemNameServer(domain)
	if err != nil {
		return err
	}

	zone, err2 := api.getZone(domain)

	if err2 != nil {
		return err2
	}

	zone.Origin = domain
	zone.SystemNameServer = systemNameServer.Name

	zone.IncludeWwwForMain = false

	zone.Soa.TTL = zoneTTL

	// empty out NameServers and ResourceRecords, add what it should be
	zone.NameServers = []*models.Nameserver{}
	zone.ResourceRecords = []*ResourceRecord{}

	zone.ResourceRecords = append(zone.ResourceRecords, resourceRecords...)

	// naive approach, the first nameserver passed should be the systemNameServer, the will be named alphabetically
	sort.Slice(nameServers, func(i, j int) bool {
		return nameServers[i].Name < nameServers[j].Name
	})

	zone.NameServers = append(zone.NameServers, nameServers...)

	_, putErr := api.request("PUT", "zone/"+domain+"/"+systemNameServer.Name, zone)

	if putErr != nil {
		return putErr
	}

	return nil
}

func (api *autoDNSProvider) updateDomain(name string, domain *Domain) error {
	_, err := api.request("PUT", "domain/"+name, domain)
	if err != nil {
		return err
	}

	return nil
}

package linode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const (
	mediaType      = "application/json"
	defaultBaseURL = "https://api.linode.com/v4/"
	domainsPath    = "domains"
)

func (api *linodeProvider) fetchDomainList() error {
	api.domainIndex = map[string]int{}
	page := 1
	for {
		dr := &domainResponse{}
		endpoint := fmt.Sprintf("%s?page=%d", domainsPath, page)
		if err := api.get(endpoint, dr); err != nil {
			return fmt.Errorf("failed fetching domain list (Linode): %w", err)
		}
		for _, domain := range dr.Data {
			api.domainIndex[domain.Domain] = domain.ID
		}
		if len(dr.Data) == 0 || dr.Page >= dr.Pages {
			break
		}
		page++
	}
	return nil
}

func (api *linodeProvider) getRecords(id int) ([]domainRecord, error) {
	records := []domainRecord{}
	page := 1
	for {
		dr := &recordResponse{}
		endpoint := fmt.Sprintf("%s/%d/records?page=%d", domainsPath, id, page)
		if err := api.get(endpoint, dr); err != nil {
			return nil, fmt.Errorf("failed fetching record list (Linode): %w", err)
		}

		records = append(records, dr.Data...)

		if len(dr.Data) == 0 || dr.Page >= dr.Pages {
			break
		}
		page++
	}

	return records, nil
}

func (api *linodeProvider) createRecord(domainID int, rec *recordEditRequest) (*domainRecord, error) {
	endpoint := fmt.Sprintf("%s/%d/records", domainsPath, domainID)

	req, err := api.newRequest(http.MethodPost, endpoint, rec)
	if err != nil {
		return nil, err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, api.handleErrors(resp)
	}

	record := &domainRecord{}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(record); err != nil {
		return nil, err
	}

	return record, nil
}

func (api *linodeProvider) modifyRecord(domainID, recordID int, rec *recordEditRequest) error {
	endpoint := fmt.Sprintf("%s/%d/records/%d", domainsPath, domainID, recordID)

	req, err := api.newRequest(http.MethodPut, endpoint, rec)
	if err != nil {
		return err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return api.handleErrors(resp)
	}

	return nil
}

func (api *linodeProvider) deleteRecord(domainID, recordID int) error {
	endpoint := fmt.Sprintf("%s/%d/records/%d", domainsPath, domainID, recordID)
	req, err := api.newRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return api.handleErrors(resp)
	}

	return nil
}

func (api *linodeProvider) newRequest(method, endpoint string, body any) (*http.Request, error) {
	rel, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	u := api.baseURL.ResolveReference(rel)

	buf := new(bytes.Buffer)
	if body != nil {
		err = json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", mediaType)
	req.Header.Add("Accept", mediaType)
	return req, nil
}

func (api *linodeProvider) get(endpoint string, target any) error {
	req, err := api.newRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := api.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return api.handleErrors(resp)
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(target)
}

func (api *linodeProvider) handleErrors(resp *http.Response) error {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	errs := &errorResponse{}

	if err := decoder.Decode(errs); err != nil {
		return fmt.Errorf("bad status code from Linode: %d not 200. Failed to decode response", resp.StatusCode)
	}

	buf := bytes.NewBufferString(fmt.Sprintf("bad status code from Linode: %d not 200", resp.StatusCode))

	for _, err := range errs.Errors {
		buf.WriteString("\n- ")

		if err.Field != "" {
			buf.WriteString(err.Field)
			buf.WriteString(": ")
		}

		buf.WriteString(err.Reason)
	}

	return errors.New(buf.String())
}

type basicResponse struct {
	Results int `json:"results"`
	Pages   int `json:"pages"`
	Page    int `json:"page"`
}

type domainResponse struct {
	basicResponse
	Data []struct {
		ID     int    `json:"id"`
		Domain string `json:"domain"`
	} `json:"data"`
}

type recordResponse struct {
	basicResponse
	Data []domainRecord `json:"data"`
}

type domainRecord struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Target   string `json:"target"`
	Priority uint16 `json:"priority"`
	Weight   uint16 `json:"weight"`
	Port     uint16 `json:"port"`
	Service  string `json:"service"`
	Protocol string `json:"protocol"`
	Tag      string `json:"tag"`
	TTLSec   uint32 `json:"ttl_sec"`
}

type recordEditRequest struct {
	Type     string `json:"type,omitempty"`
	Name     string `json:"name"`
	Target   string `json:"target"`
	Priority int    `json:"priority,omitempty"`
	Weight   int    `json:"weight,omitempty"`
	Port     int    `json:"port,omitempty"`
	Service  string `json:"service,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Tag      string `json:"tag,omitempty"`
	TTL      int    `json:"ttl_sec,omitempty"`
}

type errorResponse struct {
	Errors []struct {
		Field  string `json:"field"`
		Reason string `json:"reason"`
	} `json:"errors"`
}

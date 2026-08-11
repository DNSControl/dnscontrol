package desec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

const apiBase = "https://desec.io/api/v1"

// Api layer for desec.
type desecProvider struct {
	domainIndex     map[string]uint32 // stores the minimum ttl of each domain. (key = domain and value = ttl)
	domainIndexLock sync.Mutex
	token           string
}

type domainObject struct {
	Created    time.Time   `json:"created"`
	Keys       []dnssecKey `json:"keys,omitempty"`
	MinimumTTL uint32      `json:"minimum_ttl,omitempty"`
	Name       string      `json:"name,omitempty"`
	Published  time.Time   `json:"published"`
}

type resourceRecord struct {
	Subname string   `json:"subname"`
	Records []string `json:"records"`
	TTL     uint32   `json:"ttl,omitempty"`
	Type    string   `json:"type"`
	Target  string   `json:"-"`
}

type rrResponse struct {
	resourceRecord
	Created time.Time `json:"created"`
	Domain  string    `json:"domain"`
	Name    string    `json:"name"`
}

type dnssecKey struct {
	Dnskey  string   `json:"dnskey"`
	Ds      []string `json:"ds"`
	Flags   int      `json:"flags"`
	Keytype string   `json:"keytype"`
}

type errorResponse struct {
	Detail string `json:"detail"`
}
type nonFieldError struct {
	Errors []string `json:"non_field_errors"`
}

// withDomainIndex checks if the domain index is initialized. If not, it's fetched from the deSEC API.
// Next, the provided readFn function is executed to extract data from the domain index.
func (c *desecProvider) withDomainIndex(readFn func(domainIndex map[string]uint32)) error {
	// Lock index
	c.domainIndexLock.Lock()
	defer c.domainIndexLock.Unlock()

	// Init index if needed
	if c.domainIndex == nil {
		printer.Debugf("Domain index not yet populated, fetching now\n")
		var err error
		c.domainIndex, err = c.fetchDomainIndex()
		if err != nil {
			return fmt.Errorf("failed to fetch domain index: %w", err)
		}
	}

	// Execute handler on index
	readFn(c.domainIndex)
	return nil
}

// listDomainIndex lists all the available domains in the domain index.
func (c *desecProvider) listDomainIndex() (domains []string, err error) {
	err = c.withDomainIndex(func(domainIndex map[string]uint32) {
		domains = make([]string, 0, len(domainIndex))
		for domain := range domainIndex {
			domains = append(domains, domain)
		}
	})
	return
}

// searchDomainIndex performs a lookup to the domain index for the TTL of the domain.
func (c *desecProvider) searchDomainIndex(domain string) (ttl uint32, found bool, err error) {
	err = c.withDomainIndex(func(domainIndex map[string]uint32) {
		ttl, found = domainIndex[domain]
	})
	return
}

func (c *desecProvider) fetchDomainIndex() (map[string]uint32, error) {
	endpoint := "/domains/"
	var domainIndex map[string]uint32
	bodyString, resp, err := c.get(endpoint, "GET")
	if resp.StatusCode == http.StatusBadRequest && resp.Header.Get("Link") != "" {
		// pagination is required
		links := convertLinks(resp.Header.Get("Link"))
		endpoint = links["first"]
		printer.Debugf("initial endpoint %s\n", endpoint)
		for endpoint != "" {
			bodyString, resp, err = c.get(endpoint, "GET")
			if err != nil {
				return nil, fmt.Errorf("failed fetching domains: %w", err)
			}
			domainIndex, err = appendDomainIndexFromResponse(domainIndex, bodyString)
			if err != nil {
				return nil, fmt.Errorf("failed fetching domains: %w", err)
			}
			links = convertLinks(resp.Header.Get("Link"))
			endpoint = links["next"]
			printer.Debugf("next endpoint %s\n", endpoint)
		}
		printer.Debugf("Domain Index fetched with pagination (%d domains)\n", len(domainIndex))
		return domainIndex, nil // domainIndex was build using pagination without errors
	}

	// no pagination required
	if err != nil && resp.StatusCode != http.StatusBadRequest {
		return nil, fmt.Errorf("failed fetching domains: %w", err)
	}
	domainIndex, err = appendDomainIndexFromResponse(domainIndex, bodyString)
	if err != nil {
		return nil, err
	}
	printer.Debugf("Domain Index fetched without pagination (%d domains)\n", len(domainIndex))
	return domainIndex, nil
}

func appendDomainIndexFromResponse(domainIndex map[string]uint32, bodyString []byte) (map[string]uint32, error) {
	var dr []domainObject
	err := json.Unmarshal(bodyString, &dr)
	if err != nil {
		return nil, err
	}

	if domainIndex == nil {
		domainIndex = make(map[string]uint32, len(dr))
	}
	for _, domain := range dr {
		// deSEC allows different minimum ttls per domain
		// we store the actual minimum ttl to use it in desecProvider.go GetDomainCorrections() to enforce the minimum ttl and avoid api errors.
		domainIndex[domain.Name] = domain.MinimumTTL
	}
	return domainIndex, nil
}

// Parses the Link Header into a map (https://github.com/desec-io/desec-tools/blob/main/fetch_zone.py#L13)
func convertLinks(links string) map[string]string {
	mapping := make(map[string]string)
	printer.Debugf("Header: %s\n", links)
	for link := range strings.SplitSeq(links, ", ") {
		tmpurl := strings.Split(link, "; ")
		if len(tmpurl) != 2 {
			printer.Printf("unexpected link header %s", link)
			continue
		}
		r := regexp.MustCompile(`rel="(.*)"`)
		matches := r.FindStringSubmatch(tmpurl[1])
		if len(matches) != 2 {
			printer.Printf("unexpected label %s", tmpurl[1])
			continue
		}
		// mapping["$label"] = "$URL"
		// URL = https://desec.io/api/v1/domains/{domain}/rrsets/?cursor=:next_cursor
		mapping[matches[1]] = strings.TrimSuffix(strings.TrimPrefix(tmpurl[0], "<"), ">")
	}
	return mapping
}

func (c *desecProvider) getRecords(domain string) ([]resourceRecord, error) {
	endpoint := "/domains/%s/rrsets/"
	var rrsNew []resourceRecord
	bodyString, resp, err := c.get(fmt.Sprintf(endpoint, domain), "GET")
	if resp.StatusCode == http.StatusBadRequest && resp.Header.Get("Link") != "" {
		// pagination required
		links := convertLinks(resp.Header.Get("Link"))
		endpoint = links["first"]
		printer.Debugf("getRecords: initial endpoint %s\n", fmt.Sprintf(endpoint, domain))
		for endpoint != "" {
			bodyString, resp, err = c.get(endpoint, "GET")
			if err != nil {
				if resp.StatusCode == http.StatusNotFound {
					return rrsNew, nil
				}
				return rrsNew, fmt.Errorf("getRecords: failed fetching rrsets: %w", err)
			}
			tmp, err := generateRRSETfromResponse(bodyString)
			if err != nil {
				return rrsNew, fmt.Errorf("failed fetching records for domain %s (deSEC): %w", domain, err)
			}
			rrsNew = append(rrsNew, tmp...)
			links = convertLinks(resp.Header.Get("Link"))
			endpoint = links["next"]
			printer.Debugf("getRecords: next endpoint %s\n", endpoint)
		}
		printer.Debugf("Build rrset using pagination (%d rrs)\n", len(rrsNew))
		return rrsNew, nil // domainIndex was build using pagination without errors
	}
	// no pagination
	if err != nil {
		return rrsNew, fmt.Errorf("failed fetching records for domain %s (deSEC): %w", domain, err)
	}
	tmp, err := generateRRSETfromResponse(bodyString)
	if err != nil {
		return rrsNew, err
	}
	rrsNew = append(rrsNew, tmp...)
	printer.Debugf("Build rrset without pagination (%d rrs)\n", len(rrsNew))
	return rrsNew, nil
}

// generateRRSETfromResponse takes the response rrset api calls and returns []resourceRecord.
func generateRRSETfromResponse(bodyString []byte) ([]resourceRecord, error) {
	var rrs []rrResponse
	var rrsNew []resourceRecord
	err := json.Unmarshal(bodyString, &rrs)
	if err != nil {
		return rrsNew, err
	}
	// deSEC returns round robin records as array but dnsconfig expects single entries for each record
	// we will create one object per record except of TXT records which are handled as array of string by dnscontrol aswell.
	for i := range rrs {
		tmp := resourceRecord{
			TTL:     rrs[i].TTL,
			Type:    rrs[i].Type,
			Subname: rrs[i].Subname,
			Records: rrs[i].Records,
		}
		rrsNew = append(rrsNew, tmp)
	}
	return rrsNew, nil
}

func (c *desecProvider) createDomain(domain string) error {
	endpoint := "/domains/"
	pl := domainObject{Name: domain}
	byt, _ := json.Marshal(pl)
	var resp []byte
	var err error
	if resp, err = c.post(endpoint, "POST", byt); err != nil {
		return fmt.Errorf("failed domain create (deSEC): %w", err)
	}
	dm := domainObject{}
	err = json.Unmarshal(resp, &dm)
	if err != nil {
		return err
	}
	printer.Printf("To enable DNSSEC validation for your domain, make sure to convey the DS record(s) to your registrar:\n")
	for _, key := range dm.Keys {
		printer.Printf("DNSKEY: %s\n", key.Dnskey)
		printer.Printf("DS record(s):\n")
		for _, d := range key.Ds {
			printer.Printf("  %s\n", d)
		}
	}
	c.domainIndexLock.Lock()
	defer c.domainIndexLock.Unlock()
	if c.domainIndex != nil {
		c.domainIndex[domain] = dm.MinimumTTL
	}
	return nil
}

// upsertRR will create or override the RRSet with the provided resource record.
func (c *desecProvider) upsertRR(rr []resourceRecord, domain string) error {
	endpoint := fmt.Sprintf("/domains/%s/rrsets/", domain)
	byt, _ := json.Marshal(rr)
	if _, err := c.post(endpoint, "PUT", byt); err != nil {
		return fmt.Errorf("failed create RRset (deSEC): %w", err)
	}
	return nil
}

// Uncomment this function in case of using it
// It was commented out to satisfy `staticcheck` warnings about unused code
// func (c *desecProvider) deleteRR(domain, shortname, t string) error {
//	endpoint := fmt.Sprintf("/domains/%s/rrsets/%s/%s/", domain, shortname, t)
//	if _, _, err := c.get(endpoint, "DELETE"); err != nil {
//		return fmt.Errorf("failed delete RRset (deSEC): %w", err)
//	}
//	return nil
//}

func (c *desecProvider) get(target, method string) ([]byte, *http.Response, error) {
	retrycnt := 0
	var endpoint string
	if strings.Contains(target, "http") {
		endpoint = target
	} else {
		endpoint = apiBase + target
	}
retry:
	client := &http.Client{}
	req, _ := http.NewRequest(method, endpoint, nil)
	q := req.URL.Query()
	req.Header.Add("Authorization", "Token "+c.token)

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, resp, err
	}

	bodyString, _ := io.ReadAll(resp.Body)
	// Got error from API ?
	if resp.StatusCode > 299 {
		if resp.StatusCode == http.StatusTooManyRequests && retrycnt < 5 {
			retrycnt++
			// we've got rate limiting and will try to get the Retry-After Header if this fails we fallback to sleep for 500ms max. 5 retries.
			waitfor := resp.Header.Get("Retry-After")
			if waitfor != "" {
				wait, err := strconv.ParseInt(waitfor, 10, 64)
				if err == nil {
					if wait > 180 {
						return []byte{}, resp, errors.New("rate limiting exceeded")
					}
					printer.Warnf("Rate limiting.. waiting for %s seconds\n", waitfor)
					time.Sleep(time.Duration(wait+1) * time.Second)
					goto retry
				}
			}
			printer.Warnf("Rate limiting.. waiting for 500 milliseconds\n")
			time.Sleep(500 * time.Millisecond)
			goto retry
		}
		var errResp errorResponse
		var nfieldErrors []nonFieldError
		err = json.Unmarshal(bodyString, &errResp)
		if err == nil {
			return bodyString, resp, errors.New(errResp.Detail)
		}
		err = json.Unmarshal(bodyString, &nfieldErrors)
		if err == nil && len(nfieldErrors) > 0 {
			if len(nfieldErrors[0].Errors) > 0 {
				return bodyString, resp, errors.New(nfieldErrors[0].Errors[0])
			}
		}
		return bodyString, resp, fmt.Errorf("HTTP status %s Body: %s, the API does not provide more information", resp.Status, bodyString)
	}
	return bodyString, resp, nil
}

func (c *desecProvider) post(target, method string, payload []byte) ([]byte, error) {
	retrycnt := 0
	var endpoint string
	if strings.Contains(target, "http") {
		endpoint = target
	} else {
		endpoint = apiBase + target
	}
retry:
	client := &http.Client{}
	req, err := http.NewRequest(method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return []byte{}, err
	}
	q := req.URL.Query()
	if endpoint != "/auth/login/" {
		req.Header.Add("Authorization", "Token "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}

	bodyString, _ := io.ReadAll(resp.Body)

	// Got error from API ?
	if resp.StatusCode > 299 {
		if resp.StatusCode == http.StatusTooManyRequests && retrycnt < 5 {
			retrycnt++
			// we've got rate limiting and will try to get the Retry-After Header if this fails we fallback to sleep for 500ms max. 5 retries.
			waitfor := resp.Header.Get("Retry-After")
			if waitfor != "" {
				wait, err := strconv.ParseInt(waitfor, 10, 64)
				if err == nil {
					if wait > 180 {
						return []byte{}, errors.New("rate limiting exceeded")
					}
					printer.Warnf("Rate limiting.. waiting for %s seconds\n", waitfor)
					time.Sleep(time.Duration(wait+1) * time.Second)
					goto retry
				}
			}
			printer.Warnf("Rate limiting.. waiting for 500 milliseconds\n")
			time.Sleep(500 * time.Millisecond)
			goto retry
		}
		var errResp errorResponse
		var nfieldErrors []nonFieldError
		err = json.Unmarshal(bodyString, &errResp)
		if err == nil {
			return bodyString, fmt.Errorf("HTTP status %d %s details: %s", resp.StatusCode, resp.Status, errResp.Detail)
		}
		err = json.Unmarshal(bodyString, &nfieldErrors)
		if err == nil && len(nfieldErrors) > 0 {
			if len(nfieldErrors[0].Errors) > 0 {
				return bodyString, errors.New(nfieldErrors[0].Errors[0])
			}
		}
		return bodyString, fmt.Errorf("HTTP status %s Body: %s, the API does not provide more information", resp.Status, bodyString)
	}
	// time.Sleep(334 * time.Millisecond)
	return bodyString, nil
}

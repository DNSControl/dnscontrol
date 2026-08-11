package adguardhome

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

type adguardHomeProvider struct {
	username string
	password string
	host     string
}

type requestParams map[string]any

type errorResponse struct {
	Message string `json:"message"`
}

type rewriteEntry struct {
	Domain string `json:"domain"`
	Answer string `json:"answer"`
}

func (c *adguardHomeProvider) write(method, endpoint string, params requestParams) ([]byte, error) {
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(c.username+":"+c.password))

	reqBodyJSON, err := json.Marshal(params)
	if err != nil {
		return []byte{}, err
	}

	client := &http.Client{}
	req, _ := http.NewRequest(method, c.host+endpoint, bytes.NewBuffer(reqBodyJSON))
	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Content-Type", "application/json")

	retrycnt := 0

retry:
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}

	bodyString, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		retrycnt++
		if retrycnt == 5 {
			return bodyString, errors.New("rate limit exceeded")
		}
		printer.Warnf("rate limiting.. waiting for %d second(s)\n", retrycnt*10)
		time.Sleep(time.Second * time.Duration(retrycnt*10))
		goto retry
	}

	var errResp errorResponse
	err = json.Unmarshal(bodyString, &errResp)
	if err == nil {
		return bodyString, fmt.Errorf("AdguardHome API error: %s URL:%s%s ", errResp.Message, req.Host, req.URL.RequestURI())
	}

	if resp.StatusCode == http.StatusOK {
		return bodyString, nil
	}
	return nil, errors.New(string(bodyString))
}

func (c *adguardHomeProvider) get(endpoint string) ([]byte, error) {
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(c.username+":"+c.password))

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, c.host+endpoint, nil)
	req.Header.Add("Authorization", authHeader)

	retrycnt := 0

retry:
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}

	bodyString, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		retrycnt++
		if retrycnt == 5 {
			return bodyString, errors.New("rate limit exceeded")
		}
		printer.Warnf("rate limiting.. waiting for %d second(s)\n", retrycnt*10)
		time.Sleep(time.Second * time.Duration(retrycnt*10))
		goto retry
	}

	var errResp errorResponse
	err = json.Unmarshal(bodyString, &errResp)
	if err == nil {
		return bodyString, fmt.Errorf("AdguardHome API error: %s URL:%s%s ", errResp.Message, req.Host, req.URL.RequestURI())
	}

	if resp.StatusCode == http.StatusOK {
		return bodyString, nil
	}
	return nil, errors.New(string(bodyString))
}

func (c *adguardHomeProvider) createRecord(r rewriteEntry) error {
	rec := requestParams{
		"domain": r.Domain,
		"answer": r.Answer,
	}

	if _, err := c.write(http.MethodPost, "/control/rewrite/add", rec); err != nil {
		return fmt.Errorf("failed to create record (adguard home): %w", err)
	}
	return nil
}

func (c *adguardHomeProvider) deleteRecord(r rewriteEntry) error {
	rec := requestParams{
		"domain": r.Domain,
		"answer": r.Answer,
	}
	if _, err := c.write(http.MethodPost, "/control/rewrite/delete", rec); err != nil {
		return fmt.Errorf("failed to delete record (adguard home): %w", err)
	}
	return nil
}

func (c *adguardHomeProvider) modifyRecord(oldRe, newRe rewriteEntry) error {
	rec := requestParams{
		"target": oldRe,
		"update": newRe,
	}

	if _, err := c.write(http.MethodPut, "/control/rewrite/update", rec); err != nil {
		return fmt.Errorf("failed to update record (adguard home): %w", err)
	}
	return nil
}

func (c *adguardHomeProvider) getRecords(domain string) ([]rewriteEntry, error) {
	bodyString, err := c.get("/control/rewrite/list")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch records from adguardhome: %w", err)
	}

	var resp []rewriteEntry
	err = json.Unmarshal(bodyString, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse records list from adguardhome: %w", err)
	}

	records := make([]rewriteEntry, 0, len(resp))
	for _, r := range resp {
		if !strings.HasSuffix(r.Domain, "."+domain) && r.Domain != domain {
			continue
		}
		records = append(records, r)
	}

	return records, nil
}

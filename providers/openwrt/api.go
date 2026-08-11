package openwrt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

var idCounter uint = 0

type rpcRequest struct {
	ID     uint   `json:"id"`
	Method string `json:"method"`
	Params []any  `json:"params"`
}

func getAuthorization(username string, password string, host string) (string, error) {
	idCounter += 1
	reqBody, err := json.Marshal(rpcRequest{
		ID:     idCounter,
		Method: "login",
		Params: []any{
			username,
			password,
		},
	})
	if err != nil {
		return "", err
	}

	client := &http.Client{}

	req, _ := http.NewRequest(
		http.MethodGet,
		host+"/cgi-bin/luci/rpc/auth",
		bytes.NewBuffer(reqBody),
	)

	req.Header.Add("Content-Type", "application/json")

	retryCount := 0

retry:
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	bodyString, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		retryCount++
		if retryCount == 5 {
			return string(bodyString), errors.New("rate limit exceeded")
		}
		printer.Warnf("rate limiting.. waiting for %d second(s)\n", retryCount*10)
		time.Sleep(time.Second * time.Duration(retryCount*10))
		goto retry
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(string(bodyString))
	}

	var response struct {
		ID     int    `json:"id"`
		Result string `json:"result"`
		Error  error  `json:"error"`
	}
	err = json.Unmarshal(bodyString, &response)
	if err != nil {
		return "", err
	}

	if response.Result == "" {
		return "", fmt.Errorf("authentication token is empty")
	} else {
		return response.Result, response.Error
	}
}

func (c *openwrtProvider) uciApply() ([]any, error) {
	resp, err := c.uciCall("apply", []any{})
	if err != nil {
		return nil, err
	}

	var response struct {
		ID     int   `json:"id"`
		Result []any `json:"result"`
		Error  error `json:"error"`
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}

	return response.Result, response.Error
}

func (c *openwrtProvider) uciSection(sectionType string, values nativeRecord) (bool, error) {
	resp, err := c.uciCall("section", []any{"dhcp", sectionType, nil, values})
	if err != nil {
		return false, err
	}

	var response struct {
		ID     int   `json:"id"`
		Result bool  `json:"result"`
		Error  error `json:"error"`
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return false, err
	}

	if !response.Result {
		return false, errors.New("failed to create record")
	}

	return response.Result, response.Error
}

func (c *openwrtProvider) uciDelete(section string) (bool, error) {
	resp, err := c.uciCall("delete", []any{"dhcp", section})
	if err != nil {
		return false, err
	}

	var response struct {
		ID     int   `json:"id"`
		Result bool  `json:"result"`
		Error  error `json:"error"`
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return false, err
	}

	if !response.Result {
		return false, errors.New("failed to delete record")
	}

	return response.Result, response.Error
}

func (c *openwrtProvider) uciTset(section string, values nativeRecord) (bool, error) {
	resp, err := c.uciCall("tset", []any{"dhcp", section, values})
	if err != nil {
		return false, err
	}

	var response struct {
		ID     int   `json:"id"`
		Result bool  `json:"result"`
		Error  error `json:"error"`
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return false, err
	}

	if !response.Result {
		return false, errors.New("failed to modify record")
	}

	return response.Result, response.Error
}

func (c *openwrtProvider) uciGetAll() (map[string]nativeRecord, error) {
	resp, err := c.uciCall("get_all", []any{"dhcp"})
	if err != nil {
		return nil, err
	}

	var response struct {
		ID     int                     `json:"id"`
		Result map[string]nativeRecord `json:"result"`
		Error  error                   `json:"error"`
	}
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}

	return response.Result, response.Error
}

func (c *openwrtProvider) uciCall(method string, params []any) ([]byte, error) {
	client := &http.Client{}

	idCounter += 1
	requestBody, err := json.Marshal(rpcRequest{
		ID:     idCounter,
		Method: method,
		Params: params,
	})
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest(
		http.MethodGet,
		c.host+"/cgi-bin/luci/rpc/uci?auth="+c.auth,
		bytes.NewReader(requestBody),
	)

	req.Header.Add("Content-Type", "application/json")

	retryCount := 0

retry:
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	bodyString, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		retryCount++
		if retryCount == 5 {
			return bodyString, errors.New("rate limit exceeded")
		}
		printer.Warnf("rate limiting.. waiting for %d second(s)\n", retryCount*10)
		time.Sleep(time.Second * time.Duration(retryCount*10))
		goto retry
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(bodyString))
	}

	return bodyString, nil
}

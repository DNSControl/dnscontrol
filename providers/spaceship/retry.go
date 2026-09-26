package spaceship

import (
	"context"
	"errors"
	"time"

	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/namecheap/go-spaceship-sdk/client"
)

const (
	// defaultRetryAfter is how long to wait after a 429 that carries no
	// Retry-After header. Spaceship's documented windows are 300s; a missing
	// header is treated as a short pause rather than the full window.
	defaultRetryAfter = 30 * time.Second

	// retryAfterMargin is added to a server-supplied Retry-After so the next
	// attempt lands after the window reset rather than on it.
	retryAfterMargin = time.Second

	// maxRateLimitWait bounds the total time one request may spend waiting
	// out 429s. Two full 300s Spaceship windows plus slack. A push is not
	// transactional, so giving up halfway leaves the zone half-updated.
	maxRateLimitWait = 10 * time.Minute
)

func (c *spaceshipProvider) withRetry(f func() error) error {
	sleep := c.sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	var waited time.Duration
	for {
		err := f()
		if err == nil || !client.IsRateLimitError(err) {
			return err
		}

		wait := defaultRetryAfter
		var apiErr *client.SpaceshipApiError
		if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
			wait = apiErr.RetryAfter + retryAfterMargin
		}
		if waited+wait > maxRateLimitWait {
			return err
		}

		printer.Printf("SPACESHIP: rate limited, retrying in %v\n", wait)
		sleep(wait)
		waited += wait
	}
}

func (c *spaceshipProvider) getDomainInfo(domain string) (client.DomainInfo, error) {
	var info client.DomainInfo
	err := c.withRetry(func() error {
		var err error
		info, err = c.client.GetDomainInfo(context.Background(), domain)
		return err
	})
	return info, err
}

func (c *spaceshipProvider) getDomainList() (client.DomainList, error) {
	var list client.DomainList
	err := c.withRetry(func() error {
		var err error
		list, err = c.client.GetDomainList(context.Background())
		return err
	})
	return list, err
}

func (c *spaceshipProvider) getDNSRecords(domain string) ([]client.DNSRecord, error) {
	var records []client.DNSRecord
	err := c.withRetry(func() error {
		var err error
		records, err = c.client.GetDNSRecords(context.Background(), domain)
		return err
	})
	return records, err
}

func (c *spaceshipProvider) upsertRecords(domain string, records []client.DNSRecord) error {
	return c.withRetry(func() error {
		return c.client.UpsertDNSRecords(context.Background(), domain, true, records)
	})
}

func (c *spaceshipProvider) deleteRecords(domain string, records []client.DNSRecord) error {
	return c.withRetry(func() error {
		return c.client.DeleteDNSRecords(context.Background(), domain, records)
	})
}

func (c *spaceshipProvider) updateDomainNameServers(domain string, req client.UpdateNameserverRequest) error {
	return c.withRetry(func() error {
		return c.client.UpdateDomainNameServers(context.Background(), domain, req)
	})
}

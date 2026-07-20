package alidns

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
)

func (a *aliDNSDsp) getDomainVersionInfo(domain string) (*domainVersionInfo, error) {
	// Check cache first
	a.cacheMu.Lock()
	info, ok := a.domainVersionCache[domain]
	a.cacheMu.Unlock()
	if ok {
		return info, nil
	}

	req := alidns.CreateDescribeDomainInfoRequest()
	req.DomainName = domain

	resp, err := a.client.DescribeDomainInfo(req)
	if err != nil {
		return nil, err
	}

	// Determine minTTL based on VersionCode
	var minTTL uint32
	switch resp.VersionCode {
	case "version_enterprise_advanced":
		minTTL = 1 // Enterprise Ultimate Edition
	case "version_personal", "mianfei":
		minTTL = 600 // Personal Edition and Free Edition
	default:
		// Use MinTtl from API if available, otherwise default to 600
		if resp.MinTtl > 0 {
			minTTL = uint32(resp.MinTtl)
		} else {
			minTTL = 600
		}
	}

	info = &domainVersionInfo{
		versionCode: resp.VersionCode,
		minTTL:      minTTL,
		maxTTL:      86400,
	}
	a.cacheMu.Lock()
	a.domainVersionCache[domain] = info
	a.cacheMu.Unlock()
	return info, nil
}

// GetNameservers returns the nameservers for a domain.
func (a *aliDNSDsp) getNameservers(domain string) ([]string, error) {
	req := alidns.CreateDescribeDomainInfoRequest()
	req.DomainName = domain

	resp, err := a.client.DescribeDomainInfo(req)
	if err != nil {
		return nil, err
	}

	// Add trailing dot to each nameserver to make them FQDNs
	nameservers := make([]string, len(resp.DnsServers.DnsServer))
	for i, ns := range resp.DnsServers.DnsServer {
		if ns != "" && ns[len(ns)-1] != '.' {
			nameservers[i] = ns + "."
		} else {
			nameservers[i] = ns
		}
	}

	return nameservers, nil
}

func (a *aliDNSDsp) deleteRecordset(records []*models.RecordConfig, domainName string) error {
	for _, r := range records {
		req := alidns.CreateDeleteDomainRecordRequest()
		original, ok := r.Original.(*alidns.Record)
		if !ok {
			return fmt.Errorf("deleteRecordset: record original is not of type *alidns.Record")
		}
		req.RecordId = original.RecordId

		_, err := a.client.DeleteDomainRecord(req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *aliDNSDsp) createRecordset(records []*models.RecordConfig, domainName string) error {
	for _, r := range records {
		req := alidns.CreateAddDomainRecordRequest()
		req.DomainName = domainName
		req.RR = r.Name
		req.Type = r.Type
		req.TTL = requests.Integer(fmt.Sprintf("%d", r.TTL))
		req.Value = recordToNativeContent(r)

		// Set priority for MX and SRV records
		if r.Type == "MX" || r.Type == "SRV" {
			req.Priority = requests.Integer(fmt.Sprintf("%d", recordToNativePriority(r)))
		}

		_, err := a.client.AddDomainRecord(req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *aliDNSDsp) updateRecordset(existing, desired []*models.RecordConfig, domainName string) error {
	// Strategy: Delete all existing records, then create all desired records.
	// This is the simplest and most reliable approach because:
	// 1. The number of records in a recordset may change
	// 2. There's no guaranteed 1:1 mapping between existing and desired records
	// 3. Alibaba Cloud API requires RecordId for updates, which desired records don't have

	// Delete all existing records first
	if err := a.deleteRecordset(existing, domainName); err != nil {
		return err
	}

	// Then create all desired records
	return a.createRecordset(desired, domainName)
}

// describeDomainRecordsAll fetches all domain records for 'domain', handling
// pagination transparently. It returns the slice of *alidns.Record or an error.
func (a *aliDNSDsp) describeDomainRecordsAll(domain string) ([]*alidns.Record, error) {
	// The SDK returns a slice of value Records (not pointers). We fetch pages
	// as values and then convert to pointers before returning.
	fetch := func(pageNumber, pageSize int) ([]alidns.Record, int, error) {
		req := alidns.CreateDescribeDomainRecordsRequest()
		req.Status = "Enable"
		req.DomainName = domain
		req.PageNumber = requests.NewInteger(pageNumber)
		req.PageSize = requests.NewInteger(pageSize)

		resp, err := a.client.DescribeDomainRecords(req)
		if err != nil {
			return nil, 0, err
		}

		total := int(resp.TotalCount)
		return resp.DomainRecords.Record, total, nil
	}

	vals, err := paginateAll(fetch, 500)
	if err != nil {
		return nil, err
	}
	out := make([]*alidns.Record, 0, len(vals))
	for i := range vals {
		out = append(out, &vals[i])
	}
	return out, nil
}

func (a *aliDNSDsp) describeDomainsAll() ([]string, error) {
	// describeDomainsAll fetches all domains in the account, handling pagination.
	fetch := func(pageNumber, pageSize int) ([]string, int, error) {
		req := alidns.CreateDescribeDomainsRequest()
		req.PageNumber = requests.NewInteger(pageNumber)
		req.PageSize = requests.NewInteger(pageSize)

		resp, err := a.client.DescribeDomains(req)
		if err != nil {
			return nil, 0, err
		}

		domains := make([]string, 0, len(resp.Domains.Domain))
		for _, d := range resp.Domains.Domain {
			domains = append(domains, d.DomainName)
		}

		total := int(resp.TotalCount)
		return domains, total, nil
	}

	return paginateAll(fetch, 100)
}

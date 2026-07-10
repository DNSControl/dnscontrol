package tencentdns

import (
	"fmt"
	"strings"
	"time"

	intlcommon "github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/common"
	intlprofile "github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/common/profile"
	intldomain "github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/domain/v20180808"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	domain "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/domain/v20180808"
)

const (
	domainBatchPollAttempts = 30
	domainBatchPollInterval = 2 * time.Second

	intlDNSPodEndpoint = "dnspod.intl.tencentcloudapi.com"
	intlDomainEndpoint = "domain.intl.tencentcloudapi.com"
)

type tencentCloudClient struct {
	dnspodClient        *dnspod.Client
	domainClient        *domain.Client
	intlDomainClient    *intldomain.Client
	useIntlDomainClient bool
}

func newClient(secretID, secretKey, region, dnspodEndpoint string, useIntlDomainClient bool) (*tencentCloudClient, error) {
	credential := common.NewCredential(secretID, secretKey)

	dnspodProfile := profile.NewClientProfile()
	if dnspodEndpoint != "" {
		dnspodProfile.HttpProfile.Endpoint = dnspodEndpoint
	}

	dpc, err := dnspod.NewClient(credential, region, dnspodProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to create dnspod client: %w", err)
	}

	client := &tencentCloudClient{
		dnspodClient:        dpc,
		useIntlDomainClient: useIntlDomainClient,
	}

	if useIntlDomainClient {
		intlCredential := intlcommon.NewCredential(secretID, secretKey)
		intlDomainProfile := intlprofile.NewClientProfile()
		intlDomainProfile.HttpProfile.Endpoint = intlDomainEndpoint

		idc, err := intldomain.NewClient(intlCredential, region, intlDomainProfile)
		if err != nil {
			return nil, fmt.Errorf("failed to create intl domain client: %w", err)
		}
		client.intlDomainClient = idc
		return client, nil
	}

	domainProfile := profile.NewClientProfile()
	dmc, err := domain.NewClient(credential, region, domainProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to create domain client: %w", err)
	}
	client.domainClient = dmc

	return client, nil
}

func (c *tencentCloudClient) fetchRecords(domainName string) ([]*dnspod.RecordListItem, error) {
	var records []*dnspod.RecordListItem
	var offset uint64 = 0
	var limit uint64 = 1000

	for {
		request := dnspod.NewDescribeRecordListRequest()
		request.Domain = new(domainName)
		request.Offset = new(offset)
		request.Limit = new(limit)

		response, err := c.dnspodClient.DescribeRecordList(request)
		if err != nil {
			return nil, err
		}

		records = append(records, response.Response.RecordList...)

		if uint64(len(records)) >= *response.Response.RecordCountInfo.TotalCount {
			break
		}
		offset += limit
	}

	return records, nil
}

func (c *tencentCloudClient) getNameservers(domainName string) ([]string, error) {
	request := dnspod.NewDescribeDomainRequest()
	request.Domain = new(domainName)

	response, err := c.dnspodClient.DescribeDomain(request)
	if err != nil {
		return nil, err
	}

	var nss []string
	for _, ns := range response.Response.DomainInfo.DnspodNsList {
		nss = append(nss, *ns)
	}
	return nss, nil
}

func (c *tencentCloudClient) getMinTTL(domainName string) (uint32, error) {
	request := dnspod.NewDescribeDomainRequest()
	request.Domain = new(domainName)

	response, err := c.dnspodClient.DescribeDomain(request)
	if err != nil {
		return 0, err
	}
	if response.Response == nil || response.Response.DomainInfo == nil || response.Response.DomainInfo.Grade == nil {
		return defaultTTL, nil
	}
	grade := *response.Response.DomainInfo.Grade

	packageRequest := dnspod.NewDescribePackageDetailRequest()
	packageResponse, err := c.dnspodClient.DescribePackageDetail(packageRequest)
	if err != nil {
		return 0, err
	}
	if packageResponse.Response == nil {
		return defaultTTL, nil
	}

	return minTTLForGrade(grade, packageResponse.Response.Info), nil
}

func minTTLForGrade(grade string, packages []*dnspod.PackageDetailItem) uint32 {
	for _, item := range packages {
		if item.DomainGrade == nil || *item.DomainGrade != grade || item.MinTtl == nil {
			continue
		}
		return uint32(*item.MinTtl)
	}
	return defaultTTL
}

func (c *tencentCloudClient) getRegistrarNameservers(domainName string) ([]string, error) {
	request := dnspod.NewDescribeDomainWhoisRequest()
	request.Domain = new(domainName)

	response, err := c.dnspodClient.DescribeDomainWhois(request)
	if err != nil {
		return nil, err
	}

	var nss []string
	for _, ns := range response.Response.Info.NameServers {
		nss = append(nss, *ns)
	}
	return nss, nil
}

func (c *tencentCloudClient) updateRegistrarNameservers(domainName string, nss []string) error {
	if c.useIntlDomainClient {
		return c.updateIntlRegistrarNameservers(domainName, nss)
	}

	request := domain.NewModifyDomainDNSBatchRequest()
	request.Domains = common.StringPtrs([]string{domainName})
	request.Dns = common.StringPtrs(nss)

	response, err := c.domainClient.ModifyDomainDNSBatch(request)
	if err != nil {
		return err
	}
	if response.Response == nil || response.Response.LogId == nil {
		return nil
	}
	return c.waitForDomainBatch(*response.Response.LogId, domainName)
}

func (c *tencentCloudClient) updateIntlRegistrarNameservers(domainName string, nss []string) error {
	request := intldomain.NewBatchModifyIntlDomainDNSRequest()
	request.Domains = intlcommon.StringPtrs([]string{domainName})
	request.Dns = intlcommon.StringPtrs(nss)

	response, err := c.intlDomainClient.BatchModifyIntlDomainDNS(request)
	if err != nil {
		return err
	}
	if response.Response == nil || response.Response.LogId == nil {
		return nil
	}
	return c.waitForIntlDomainBatch(*response.Response.LogId, domainName)
}

func (c *tencentCloudClient) waitForDomainBatch(logID uint64, domainName string) error {
	for range domainBatchPollAttempts {
		request := domain.NewDescribeBatchOperationLogDetailsRequest()
		request.LogId = new(int64(logID))
		request.Offset = common.Int64Ptr(0)
		request.Limit = common.Int64Ptr(200)

		response, err := c.domainClient.DescribeBatchOperationLogDetails(request)
		if err != nil {
			return err
		}
		if response.Response != nil {
			status, reason, found := domainBatchStatus(response.Response.DomainBatchDetailSet, domainName)
			switch status {
			case "success":
				return nil
			case "failed":
				if reason == "" {
					reason = "unknown reason"
				}
				return fmt.Errorf("tencent domain batch operation %d failed for %s: %s", logID, domainName, reason)
			case "doing":
				// Keep polling.
			default:
				if found {
					return fmt.Errorf("tencent domain batch operation %d returned unexpected status %q for %s", logID, status, domainName)
				}
			}
		}

		time.Sleep(domainBatchPollInterval)
	}
	return fmt.Errorf("timed out waiting for tencent domain batch operation %d for %s", logID, domainName)
}

func domainBatchStatus(details []*domain.DomainBatchDetailSet, domainName string) (status, reason string, found bool) {
	for _, detail := range details {
		if detail.Domain == nil || !strings.EqualFold(*detail.Domain, domainName) {
			continue
		}
		found = true
		if detail.Status != nil {
			status = *detail.Status
		}
		if detail.Reason != nil {
			reason = *detail.Reason
		}
		return status, reason, found
	}
	return "", "", false
}

func (c *tencentCloudClient) waitForIntlDomainBatch(logID uint64, domainName string) error {
	for range domainBatchPollAttempts {
		request := intldomain.NewDescribeIntlDomainBatchDetailsRequest()
		request.LogId = new(int64(logID))
		request.Offset = intlcommon.Int64Ptr(0)
		request.Limit = intlcommon.Int64Ptr(100)

		response, err := c.intlDomainClient.DescribeIntlDomainBatchDetails(request)
		if err != nil {
			return err
		}
		if response.Response != nil {
			status, reason, found := intlDomainBatchStatus(response.Response.DomainBatchDetailSet, domainName)
			switch strings.ToLower(status) {
			case "success":
				return nil
			case "failure", "failed":
				if reason == "" {
					reason = "unknown reason"
				}
				return fmt.Errorf("tencent intl domain batch operation %d failed for %s: %s", logID, domainName, reason)
			case "", "doing":
				// Keep polling.
			default:
				if found {
					return fmt.Errorf("tencent intl domain batch operation %d returned unexpected status %q for %s", logID, status, domainName)
				}
			}
		}

		time.Sleep(domainBatchPollInterval)
	}
	return fmt.Errorf("timed out waiting for tencent intl domain batch operation %d for %s", logID, domainName)
}

func intlDomainBatchStatus(details []*intldomain.BatchDomainBuyDetails, domainName string) (status, reason string, found bool) {
	for _, detail := range details {
		if detail.Domain == nil || !strings.EqualFold(*detail.Domain, domainName) {
			continue
		}
		found = true
		if detail.Status != nil {
			status = *detail.Status
		}
		if detail.Reason != nil {
			reason = *detail.Reason
		}
		if reason == "" && detail.ReasonZh != nil {
			reason = *detail.ReasonZh
		}
		return status, reason, found
	}
	return "", "", false
}

func (c *tencentCloudClient) createRecord(domainName string, request *dnspod.CreateRecordRequest) error {
	request.Domain = new(domainName)
	_, err := c.dnspodClient.CreateRecord(request)
	return err
}

func (c *tencentCloudClient) modifyRecord(domainName string, request *dnspod.ModifyRecordRequest) error {
	request.Domain = new(domainName)
	_, err := c.dnspodClient.ModifyRecord(request)
	return err
}

func (c *tencentCloudClient) deleteRecord(domainName string, recordID uint64) error {
	request := dnspod.NewDeleteRecordRequest()
	request.Domain = new(domainName)
	request.RecordId = new(recordID)
	_, err := c.dnspodClient.DeleteRecord(request)
	return err
}

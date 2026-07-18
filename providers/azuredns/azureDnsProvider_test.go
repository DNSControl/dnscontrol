package azuredns

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	adns "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/DNSControl/dnscontrol/v4/models"
)

func TestNativeToRecordsUsesV3RecordConfig(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name       string
		azureType  string
		properties func() *adns.RecordSetProperties
		want       *models.RecordConfig
	}{
		{
			name:      "NS",
			azureType: "Microsoft.Network/dnszones/NS",
			properties: func() *adns.RecordSetProperties {
				return &adns.RecordSetProperties{NsRecords: []*adns.NsRecord{{Nsdname: new("ns1.example.net.")}}}
			},
			want: dc.MustNewRecordConfig("www", 300, dnsv2.TypeNS, "ns1.example.net."),
		},
		{
			name:      "PTR",
			azureType: "Microsoft.Network/dnszones/PTR",
			properties: func() *adns.RecordSetProperties {
				return &adns.RecordSetProperties{PtrRecords: []*adns.PtrRecord{{Ptrdname: new("host.example.net.")}}}
			},
			want: dc.MustNewRecordConfig("www", 300, dnsv2.TypePTR, "host.example.net."),
		},
		{
			name:      "empty TXT record set",
			azureType: "Microsoft.Network/dnszones/TXT",
			properties: func() *adns.RecordSetProperties {
				return &adns.RecordSetProperties{}
			},
			want: dc.MustNewRecordConfig("www", 300, dnsv2.TypeTXT, ""),
		},
		{
			name:      "segmented TXT",
			azureType: "Microsoft.Network/dnszones/TXT",
			properties: func() *adns.RecordSetProperties {
				return &adns.RecordSetProperties{TxtRecords: []*adns.TxtRecord{{Value: []*string{new("first"), new("second")}}}}
			},
			want: dc.MustNewRecordConfig("www", 300, dnsv2.TypeTXT, "firstsecond"),
		},
		{
			name:      "MX",
			azureType: "Microsoft.Network/dnszones/MX",
			properties: func() *adns.RecordSetProperties {
				return &adns.RecordSetProperties{MxRecords: []*adns.MxRecord{{Preference: new(int32(10)), Exchange: new("mail.example.net.")}}}
			},
			want: dc.MustNewRecordConfig("www", 300, dnsv2.TypeMX, uint16(10), "mail.example.net."),
		},
		{
			name:      "SRV",
			azureType: "Microsoft.Network/dnszones/SRV",
			properties: func() *adns.RecordSetProperties {
				return &adns.RecordSetProperties{SrvRecords: []*adns.SrvRecord{{Priority: new(int32(1)), Weight: new(int32(2)), Port: new(int32(443)), Target: new("service.example.net.")}}}
			},
			want: dc.MustNewRecordConfig("www", 300, dnsv2.TypeSRV, uint16(1), uint16(2), uint16(443), "service.example.net."),
		},
		{
			name:      "CAA",
			azureType: "Microsoft.Network/dnszones/CAA",
			properties: func() *adns.RecordSetProperties {
				return &adns.RecordSetProperties{CaaRecords: []*adns.CaaRecord{{Flags: new(int32(0)), Tag: new("issue"), Value: new("letsencrypt.org")}}}
			},
			want: dc.MustNewRecordConfig("www", 300, dnsv2.TypeCAA, uint8(0), "issue", "letsencrypt.org"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := tt.properties()
			properties.Fqdn = new("www.example.com.")
			properties.TTL = new(int64(300))
			set := &adns.RecordSet{Type: new(tt.azureType), Properties: properties}

			got := nativeToRecords(set, dc)
			if len(got) != 1 {
				t.Fatalf("nativeToRecords() returned %d records, want 1", len(got))
			}
			if got[0].NameFQDN != tt.want.NameFQDN || got[0].TTL != tt.want.TTL || got[0].TypeNum != tt.want.TypeNum || got[0].GetRDATA().String() != tt.want.GetRDATA().String() {
				t.Errorf("nativeToRecords() = %s %d IN %s %s, want %s %d IN %s %s", got[0].NameFQDN, got[0].TTL, got[0].Type, got[0].GetRDATA(), tt.want.NameFQDN, tt.want.TTL, tt.want.Type, tt.want.GetRDATA())
			}
			if got[0].Original != set {
				t.Error("nativeToRecords() did not preserve the original Azure record set")
			}
		})
	}
}

func TestRetryableRecordSetMutation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "rate limit",
			err:  &azcore.ResponseError{StatusCode: http.StatusTooManyRequests, ErrorCode: "TooManyRequests"},
			want: "rate-limit",
		},
		{
			name: "pending operation conflict",
			err:  newAzureResponseError(http.StatusConflict, "Conflict", azurePendingOperationConflictMessage),
			want: "pending operation",
		},
		{
			name: "wrapped pending operation conflict",
			err:  fmt.Errorf("wrapped: %w", newAzureResponseError(http.StatusConflict, "Conflict", azurePendingOperationConflictMessage)),
			want: "pending operation",
		},
		{
			name: "other conflict",
			err:  newAzureResponseError(http.StatusConflict, "Conflict", "The record set already exists."),
			want: "",
		},
		{
			name: "wrong conflict code",
			err:  newAzureResponseError(http.StatusConflict, "PreconditionFailed", azurePendingOperationConflictMessage),
			want: "",
		},
		{
			name: "non Azure error",
			err:  fmt.Errorf("plain error"),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retryableRecordSetMutation(tt.err); got != tt.want {
				t.Fatalf("retryableRecordSetMutation() = %q, want %q", got, tt.want)
			}
		})
	}
}

func newAzureResponseError(statusCode int, code string, message string) *azcore.ResponseError {
	body := fmt.Sprintf(`{"error":{"code":%q,"message":%q}}`, code, message)
	return &azcore.ResponseError{
		ErrorCode:  code,
		StatusCode: statusCode,
		RawResponse: &http.Response{
			StatusCode: statusCode,
			Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
			Body:       io.NopCloser(strings.NewReader(body)),
		},
	}
}

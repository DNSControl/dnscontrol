package azureprivatedns

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

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

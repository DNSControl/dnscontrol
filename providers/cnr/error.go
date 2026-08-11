package cnr

import (
	"fmt"

	"github.com/centralnicgroup-opensource/rtldev-middleware-go-sdk/v5/response"
)

// GetAPIError returns an error including API error code and error description.
func (n *Client) GetAPIError(format string, objectid string, r *response.Response) error {
	return fmt.Errorf(format+" %q. [%v %s]", objectid, r.GetCode(), r.GetDescription())
}

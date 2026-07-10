package netbird

import (
	"fmt"
)

func (api *netbirdProvider) listRecords(zoneID string) ([]Record, error) {
	var records []Record
	err := api.doRequest("GET", fmt.Sprintf("/dns/zones/%s/records", zoneID), nil, &records)
	return records, err
}

func (api *netbirdProvider) createRecord(zoneID string, req *CreateRecordRequest) error {
	var result Record
	return api.doRequest("POST", fmt.Sprintf("/dns/zones/%s/records", zoneID), req, &result)
}

func (api *netbirdProvider) updateRecord(zoneID string, recordID string, req *CreateRecordRequest) error {
	var result Record
	return api.doRequest("PUT", fmt.Sprintf("/dns/zones/%s/records/%s", zoneID, recordID), req, &result)
}

func (api *netbirdProvider) deleteRecord(zoneID string, recordID string) error {
	return api.doRequest("DELETE", fmt.Sprintf("/dns/zones/%s/records/%s", zoneID, recordID), nil, nil)
}

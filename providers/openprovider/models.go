package openprovider

import (
	"encoding/json"
	"strings"
)

type apiEnvelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Desc string          `json:"desc"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IP       string `json:"ip"`
}

type loginResponse struct {
	Token string `json:"token"`
}

type successResponse struct {
	Success bool `json:"success"`
}

type apiZone struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type zoneListResponse struct {
	Results []apiZone `json:"results"`
	Total   int       `json:"total"`
}

type zoneDomain struct {
	Name      string `json:"name"`
	Extension string `json:"extension"`
}

type createZoneRequest struct {
	Domain               zoneDomain  `json:"domain"`
	Type                 string      `json:"type"`
	Records              []apiRecord `json:"records"`
	IsSpamExpertsEnabled string      `json:"is_spamexperts_enabled"`
	Secured              bool        `json:"secured"`
}

// apiRecord is both the record shape returned by OpenProvider and the shape
// accepted by the zone update endpoint. OpenProvider identifies a record by
// its complete contents rather than by a stable record ID.
type apiRecord struct {
	Name  string `json:"name,omitempty"`
	Type  string `json:"type"`
	Value string `json:"value"`
	Prio  int    `json:"prio,omitempty"`
	TTL   int    `json:"ttl"`
}

// MarshalJSON keeps priority zero for record types that use the priority
// field. In particular, priority zero is valid for MX records and is required
// by OpenProvider's API; omitempty would otherwise drop it.
func (r apiRecord) MarshalJSON() ([]byte, error) {
	var prio *int
	if strings.EqualFold(r.Type, "MX") || strings.EqualFold(r.Type, "SRV") {
		prio = &r.Prio
	}
	return json.Marshal(struct {
		Name  string `json:"name,omitempty"`
		Type  string `json:"type"`
		Value string `json:"value"`
		Prio  *int   `json:"prio,omitempty"`
		TTL   int    `json:"ttl"`
	}{r.Name, r.Type, r.Value, prio, r.TTL})
}

type recordListResponse struct {
	Results []apiRecord `json:"results"`
	Total   int         `json:"total"`
}

type recordUpdate struct {
	Original apiRecord `json:"original_record"`
	Record   apiRecord `json:"record"`
}

type recordUpdates struct {
	Add    []apiRecord    `json:"add,omitempty"`
	Remove []apiRecord    `json:"remove,omitempty"`
	Update []recordUpdate `json:"update,omitempty"`
}

type updateZoneRequest struct {
	ID      int64         `json:"id"`
	Name    string        `json:"name"`
	Records recordUpdates `json:"records"`
}

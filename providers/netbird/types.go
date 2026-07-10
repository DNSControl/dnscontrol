package netbird

// zoneInfo stores cached zone information for a domain.
type zoneInfo struct {
	id     string
	domain string
}

// Zone represents a NetBird DNS zone.
type Zone struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Domain             string   `json:"domain"`
	Enabled            bool     `json:"enabled"`
	EnableSearchDomain bool     `json:"enable_search_domain"`
	DistributionGroups []string `json:"distribution_groups"`
	Records            []Record `json:"records"`
}

// Record represents a NetBird DNS record.
type Record struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

// CreateRecordRequest is used to create a new DNS record.
type CreateRecordRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

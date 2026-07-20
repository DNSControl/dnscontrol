package autodns

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/providers/bind"
)

// ResourceRecord represents DNS records in API calls.
type ResourceRecord struct {
	// The name of the record.
	// Required: true
	Name string `json:"name"`

	// Preference of the record, need for some record types e.g. MX
	// Maximum: 65535
	Pref int32 `json:"pref,omitempty"`

	// The bind notation of the record. Only used by the zone stream task!
	Raw string `json:"raw,omitempty"`

	// TTL of the record (Optionally if not set then Default SOA TTL is used)
	TTL int64 `json:"ttl,omitempty"`

	// The type of the record, e.g. A
	// Permitted values: A, AAAA, CAA, CNAME, HINFO, MX, NAPTR, NS, PTR, SRV, TXT, ALIAS
	Type string `json:"type,omitempty"`

	// The value of the record.
	Value string `json:"value,omitempty"`
}

// MainAddressRecord represents an address record in API calls.
type MainAddressRecord struct {
	// TTL of the record (Optionally if not set then Default SOA TTL is used)
	TTL int64 `json:"ttl,omitempty"`

	// The value of the record.
	Value string `json:"address,omitempty"`
}

// Zone represents the Zone in API calls.
type Zone struct {
	Origin string `json:"origin"`

	Soa *bind.SoaDefaults `json:"soa,omitempty"`

	// List of name servers
	NameServers []*models.Nameserver `json:"nameServers,omitempty"`

	// The resource records.
	// Max Items: 10000
	// Min Items: 0
	ResourceRecords []*ResourceRecord `json:"resourceRecords,omitempty"`

	// Might be set if we fetch a zone for the first time, should be migrated to ResourceRecords
	MainRecord *MainAddressRecord `json:"main,omitempty"`

	IncludeWwwForMain bool `json:"wwwInclude"`

	// Primary NameServer, needs to be passed to the system to fetch further zone info
	SystemNameServer string `json:"virtualNameServer,omitempty"`
}

// Domain represents the Domain in API calls.
// These are only present for domains where AUTODNS also is a registrar.
type Domain struct {
	Name string `json:"name,omitempty"`

	NameServers []*NameServer `json:"nameServers"`
	Zone        *Zone         `json:"zone,omitempty"`
}

// NameServer represents a nameserver in API calls.
type NameServer struct {
	// Host name of the nameserver written as a Fully-Qualified-Domain-Name (FQDN).
	Name string `json:"name"`
	// Time-to-live value of the nameservers in seconds
	TTL uint64 `json:"ttl,omitempty"`
	// IPv4 and IPv6 addresses of the name server. For GLUE records only; optional. The values for the IP addresses are only relevant for domain operations and are only used there in the case of glue name servers.
	IPAddresses []string `json:"ipAddresses,omitempty"`
}

// ListResponseMetaData represents a JSON reply to a list command.
type ListResponseMetaData struct {
	ObjectType string `json:"type"`
	ItemCount  int    `json:"summary"`
}

// JSONResponseDataZone represents the response to the DataZone call.
type JSONResponseDataZone struct {
	// The data for the response. The type of the objects are depending on the request and are also specified in the responseObject value of the response.
	Data     []*Zone               `json:"data"`
	MetaData *ListResponseMetaData `json:"object"`
}

// JSONResponseDataDomain represents the response to the DataDomain call.
type JSONResponseDataDomain struct {
	// The data for the response. The type of the objects are depending on the request and are also specified in the responseObject value of the response.
	Data []*Domain `json:"data"`
}

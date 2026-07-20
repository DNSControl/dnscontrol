package models

// DNSProvider is an interface for DNS Provider plug-ins.
type DNSProvider interface {
	GetNameservers(domain string) ([]*Nameserver, error)
	GetZoneRecords(dc *DomainConfig) (Records, error)
	GetZoneRecordsCorrections(dc *DomainConfig, existing Records) ([]*Correction, int, error)
}

// Registrar is an interface for Registrar plug-ins.
type Registrar interface {
	GetRegistrarCorrections(dc *DomainConfig) ([]*Correction, error)
}

// ProviderBase describes providers.
type ProviderBase struct {
	Name         string
	IsDefault    bool
	ProviderType string
}

// RegistrarInstance is a single registrar.
type RegistrarInstance struct {
	ProviderBase
	Driver Registrar
}

// DNSProviderInstance is a single DNS provider.
type DNSProviderInstance struct {
	ProviderBase
	Driver              DNSProvider
	NumberOfNameservers int
}

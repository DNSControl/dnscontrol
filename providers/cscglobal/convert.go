package cscglobal

// Convert the provider's native record description to models.RecordConfig.

import (
	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func recordTTL(ttl, defaultTTL uint32) uint32 {
	if ttl == 0 {
		return defaultTTL
	}
	return ttl
}

// nativeToRecordA takes an A record from DNS and returns a native RecordConfig struct.
func nativeToRecordA(nr nativeRecordA, dc *models.DomainConfig, defaultTTL uint32) *models.RecordConfig {
	return dc.MustNewRecordConfig(dc.LabelFromShort(nr.Key), recordTTL(nr.TTL, defaultTTL), dnsv2.TypeA, nr.Value)
}

// nativeToRecordCNAME takes a CNAME record from DNS and returns a native RecordConfig struct.
func nativeToRecordCNAME(nr nativeRecordCNAME, dc *models.DomainConfig, defaultTTL uint32) *models.RecordConfig {
	return dc.MustNewRecordConfig(dc.LabelFromShort(nr.Key), recordTTL(nr.TTL, defaultTTL), dnsv2.TypeCNAME, nr.Value)
}

// nativeToRecordAAAA takes an AAAA record from DNS and returns a native RecordConfig struct.
func nativeToRecordAAAA(nr nativeRecordAAAA, dc *models.DomainConfig, defaultTTL uint32) *models.RecordConfig {
	return dc.MustNewRecordConfig(dc.LabelFromShort(nr.Key), recordTTL(nr.TTL, defaultTTL), dnsv2.TypeAAAA, nr.Value)
}

// nativeToRecordTXT takes a TXT record from DNS and returns a native RecordConfig struct.
func nativeToRecordTXT(nr nativeRecordTXT, dc *models.DomainConfig, defaultTTL uint32) *models.RecordConfig {
	return dc.MustNewRecordConfig(dc.LabelFromShort(nr.Key), recordTTL(nr.TTL, defaultTTL), dnsv2.TypeTXT, nr.Value)
}

// nativeToRecordMX takes an MX record from DNS and returns a native RecordConfig struct.
func nativeToRecordMX(nr nativeRecordMX, dc *models.DomainConfig, defaultTTL uint32) *models.RecordConfig {
	return dc.MustNewRecordConfig(dc.LabelFromShort(nr.Key), recordTTL(nr.TTL, defaultTTL), dnsv2.TypeMX, nr.Priority, nr.Value)
}

// nativeToRecordNS takes a NS record from DNS and returns a native RecordConfig struct.
func nativeToRecordNS(nr nativeRecordNS, dc *models.DomainConfig, defaultTTL uint32) *models.RecordConfig {
	return dc.MustNewRecordConfig(dc.LabelFromShort(nr.Key), recordTTL(nr.TTL, defaultTTL), dnsv2.TypeNS, nr.Value)
}

// nativeToRecordSRV takes a SRV record from DNS and returns a native RecordConfig struct.
func nativeToRecordSRV(nr nativeRecordSRV, dc *models.DomainConfig, defaultTTL uint32) *models.RecordConfig {
	return dc.MustNewRecordConfig(dc.LabelFromShort(nr.Key), recordTTL(nr.TTL, defaultTTL), dnsv2.TypeSRV, nr.Priority, nr.Weight, nr.Port, nr.Value)
}

// nativeToRecordCAA takes a CAA record from DNS and returns a native RecordConfig struct.
func nativeToRecordCAA(nr nativeRecordCAA, dc *models.DomainConfig, defaultTTL uint32) *models.RecordConfig {
	return dc.MustNewRecordConfig(dc.LabelFromShort(nr.Key), recordTTL(nr.TTL, defaultTTL), dnsv2.TypeCAA, nr.Flag, nr.Tag, nr.Value)
}

package scaleway

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	domain "github.com/scaleway/scaleway-sdk-go/api/domain/v2beta1"
)

// labelFromName converts a Scaleway record `Name` (which is the short name
// relative to the zone, possibly empty for apex) to the dnscontrol label.
func labelFromName(name string) string {
	if name == "" {
		return "@"
	}
	return name
}

// nameFromLabel does the inverse for writing.
func nameFromLabel(rc *models.RecordConfig) string {
	label := rc.GetLabel()
	if label == "@" {
		return ""
	}
	return label
}

// toRecordConfig converts a Scaleway Record to a dnscontrol RecordConfig.
func toRecordConfig(dc *models.DomainConfig, r *domain.Record) (*models.RecordConfig, error) {
	label := dc.LabelFromShort(labelFromName(r.Name))
	ttl := r.TTL
	rtype := string(r.Type)
	data := strings.TrimSpace(r.Data)

	var rc *models.RecordConfig
	var err error
	switch rtype {
	case "TXT":
		// Scaleway returns the TXT value wrapped in quotes (BIND-style).
		// SetTargetTXT expects the unquoted single-string value.
		unq, unquoteErr := unquoteTXT(data)
		if unquoteErr != nil {
			return nil, unquoteErr
		}
		rc, err = dc.NewRecordConfig(label, ttl, rtype, unq)
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, data)
	}
	if err != nil {
		return nil, fmt.Errorf("SCALEWAY: unparsable %s record %q: %w", rtype, data, err)
	}
	rc.Original = r
	return rc, nil
}

// fromRecordConfig converts a dnscontrol RecordConfig to a Scaleway Record
// ready to be sent in an Add change.
func fromRecordConfig(rc *models.RecordConfig) domain.Record {
	rec := domain.Record{
		Name: nameFromLabel(rc),
		Type: domain.RecordType(rc.Type),
		TTL:  rc.TTL,
	}

	if rc.Type == "TXT" {
		// Scaleway accepts the TXT data BIND-style quoted, so build that form
		// from the joined value directly.
		rec.Data = quoteTXT(rc.GetTargetTXTJoined())
		return rec
	}
	rec.Data = rc.GetRDATA().String()
	return rec
}

// quoteTXT returns s as a BIND-style quoted string: every `\` and `"` is
// escaped with a backslash and the whole thing is wrapped in double quotes.
func quoteTXT(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' || c == '"' {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	b.WriteByte('"')
	return b.String()
}

// unquoteTXT parses a BIND-style quoted TXT value back to its raw bytes.
// If the input has no surrounding quotes, it is returned unchanged.
func unquoteTXT(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return s, nil
	}
	inner := s[1 : len(s)-1]
	var b strings.Builder
	b.Grow(len(inner))
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c == '\\' && i+1 < len(inner) {
			i++
			b.WriteByte(inner[i])
			continue
		}
		b.WriteByte(c)
	}
	return b.String(), nil
}

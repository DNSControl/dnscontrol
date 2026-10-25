package constellix

import (
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestGetZoneRecords(t *testing.T) {
	api := newFakeAPI(t)
	api.records["1"] = `[
		{"id": 11, "name": "", "type": "A", "ttl": 300, "mode": "standard", "region": "default", "ipfilter": null, "geoproximity": null, "enabled": true,
		 "value": [{"value": "192.0.2.1", "enabled": true}, {"value": "192.0.2.2", "enabled": true}]},
		{"id": 12, "name": "mx", "type": "MX", "ttl": 600,
		 "value": [{"server": "mx2", "priority": 20}]}
	]`
	p := testProvider(t, api)

	recs, err := p.GetZoneRecords(models.MustNewDomainConfig(testZone))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, rc := range recs {
		got = append(got, rc.GetLabel()+" "+rc.Type+" "+rc.ComparableV3)
	}
	want := []string{"@ A 192.0.2.1", "@ A 192.0.2.2", "mx MX 20 mx2.example.com."}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("GetZoneRecords() = %q, want %q", got, want)
	}
}

func TestGetZoneRecordsRefuses(t *testing.T) {
	tests := []struct {
		name   string
		record string
		want   string
	}{
		{
			name:   "HTTP redirect with an object value",
			record: `{"id": 1, "name": "go", "type": "HTTP", "ttl": 300, "mode": "standard", "region": "default", "enabled": true, "value": {"hard": false, "redirectType": 301, "url": "https://example.net/"}}`,
			want:   `record "go"/HTTP has a type`,
		},
		{
			name:   "failover A with an object value",
			record: `{"id": 1, "name": "fo", "type": "A", "ttl": 300, "mode": "failover", "region": "default", "enabled": true, "value": {"mode": "normal", "enabled": true, "values": [{"value": "192.0.2.1", "order": 1, "sonarCheckId": 1, "enabled": true}]}}`,
			want:   `record "fo"/A uses failover mode`,
		},
		{
			name:   "object value in standard mode",
			record: `{"id": 1, "name": "odd", "type": "A", "ttl": 300, "value": {"values": []}}`,
			want:   `record "odd"/A has a value that is not a list`,
		},
		{
			name:   "NAPTR",
			record: `{"id": 1, "name": "naptr", "type": "NAPTR", "ttl": 300, "mode": "standard", "region": "default", "enabled": true, "value": [{"order": 100, "preference": 10, "flags": "U", "service": "E2U+sip", "regularExpression": "!^.*$!sip:info@example.net!", "replacement": ".", "enabled": true}]}`,
			want:   `record "naptr"/NAPTR has a type`,
		},
		{
			name:   "non-default region",
			record: `{"id": 1, "name": "eu", "type": "A", "ttl": 300, "mode": "standard", "region": "europe", "enabled": true, "value": [{"value": "192.0.2.1", "enabled": true}]}`,
			want:   `record "eu"/A is in the GTD region "europe"`,
		},
		{
			name:   "IP filter",
			record: `{"id": 1, "name": "f", "type": "A", "ttl": 300, "mode": "standard", "region": "default", "ipfilter": {"id": 5, "name": "office"}, "enabled": true, "value": [{"value": "192.0.2.1", "enabled": true}]}`,
			want:   `record "f"/A uses an IP filter`,
		},
		{
			name:   "GeoProximity",
			record: `{"id": 1, "name": "g", "type": "A", "ttl": 300, "mode": "standard", "region": "default", "geoproximity": {"id": 3}, "enabled": true, "value": [{"value": "192.0.2.1", "enabled": true}]}`,
			want:   `record "g"/A uses GeoProximity`,
		},
		{
			name:   "disabled record",
			record: `{"id": 1, "name": "dis", "type": "A", "ttl": 300, "mode": "standard", "region": "default", "enabled": false, "value": [{"value": "192.0.2.1", "enabled": true}]}`,
			want:   `record "dis"/A is disabled`,
		},
		{
			name:   "disabled value",
			record: `{"id": 1, "name": "disv", "type": "A", "ttl": 300, "mode": "standard", "region": "default", "enabled": true, "value": [{"value": "192.0.2.1", "enabled": true}, {"value": "192.0.2.2", "enabled": false}]}`,
			want:   `record "disv"/A has a disabled value`,
		},
		{
			name:   "no values",
			record: `{"id": 1, "name": "empty", "type": "A", "ttl": 300, "mode": "standard", "region": "default", "enabled": true, "value": []}`,
			want:   `record "empty"/A has no values`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newFakeAPI(t)
			api.records["1"] = `[
				{"id": 2, "name": "www", "type": "A", "ttl": 300, "value": [{"value": "192.0.2.9", "enabled": true}]},
				` + tt.record + `]`
			p := testProvider(t, api)

			_, err := p.GetZoneRecords(models.MustNewDomainConfig(testZone))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("GetZoneRecords() error = %v, want %q", err, tt.want)
			}
			if !strings.Contains(err.Error(), "Constellix portal") {
				t.Errorf("error %q does not say where to resolve it", err)
			}
		})
	}
}

func TestGetZoneRecordsRefusesTemplateZone(t *testing.T) {
	api := newFakeAPI(t)
	api.zones[0]["template"] = map[string]any{"id": 83675283, "name": "base"}
	p := testProvider(t, api)

	_, err := p.GetZoneRecords(models.MustNewDomainConfig(testZone))
	if err == nil || !strings.Contains(err.Error(), "zone example.com is linked to a template") {
		t.Fatalf("GetZoneRecords() error = %v, want the template refusal", err)
	}
}

func TestGetZoneRecordsDefaults(t *testing.T) {
	api := newFakeAPI(t)
	// Neither the record nor its value says mode, region or enabled.
	api.records["1"] = `[{"id": 2, "name": "www", "type": "A", "ttl": 300, "value": [{"value": "192.0.2.9"}]}]`
	p := testProvider(t, api)

	recs, err := p.GetZoneRecords(models.MustNewDomainConfig(testZone))
	if err != nil || len(recs) != 1 {
		t.Fatalf("GetZoneRecords() = %v, %v; want one record", recs, err)
	}
}

// apexNS returns the apex NS records DNSControl derives from the zone's
// nameservers.
func apexNS(dc *models.DomainConfig) models.Records {
	var recs models.Records
	for _, ns := range testNameservers {
		recs = append(recs, dc.MustNewRecordConfig("@", 300, dnsv2.TypeNS, ns))
	}
	return recs
}

// corrections reads the zone from api, computes the corrections for the
// records desired returns, runs them, and returns the API writes.
func corrections(t *testing.T, api *fakeAPI, desired func(dc *models.DomainConfig) models.Records) []string {
	t.Helper()
	p := testProvider(t, api)

	dc := models.MustNewDomainConfig(testZone)
	existing, err := p.GetZoneRecords(dc)
	if err != nil {
		t.Fatal(err)
	}
	dc.Records = desired(dc)

	corrs, _, err := p.GetZoneRecordsCorrections(dc, existing)
	if err != nil {
		t.Fatal(err)
	}
	api.writes = nil
	for _, c := range corrs {
		if c.F == nil {
			continue
		}
		if err := c.F(); err != nil {
			t.Fatalf("correction %q: %v", c.Msg, err)
		}
	}
	return api.writes
}

func TestGetZoneRecordsCorrections(t *testing.T) {
	const wwwA = `{"id": 21, "name": "www", "type": "A", "ttl": 300, "value": [{"value": "192.0.2.1", "enabled": true}]}`
	const apexNSOwn = `{"id": 31, "name": "", "type": "NS", "ttl": 300, "value": [{"host": "ns11.constellix.com.", "enabled": true}]}`

	tests := []struct {
		name     string
		existing string
		desired  func(dc *models.DomainConfig) models.Records
		want     []string
	}{
		{
			name:     "create",
			existing: `[]`,
			desired: func(dc *models.DomainConfig) models.Records {
				return append(apexNS(dc), dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1"))
			},
			want: []string{`POST /domains/1/records {"name":"www","type":"A","ttl":300,"value":[{"enabled":true,"value":"192.0.2.1"}]}`},
		},
		{
			name:     "add a value",
			existing: `[` + wwwA + `]`,
			desired: func(dc *models.DomainConfig) models.Records {
				return append(apexNS(dc),
					dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.1"),
					dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.2"))
			},
			want: []string{`PUT /domains/1/records/21 {"ttl":300,"value":[{"enabled":true,"value":"192.0.2.1"},{"enabled":true,"value":"192.0.2.2"}]}`},
		},
		{
			name:     "change only the TTL",
			existing: `[` + wwwA + `]`,
			desired: func(dc *models.DomainConfig) models.Records {
				return append(apexNS(dc), dc.MustNewRecordConfig("www", 600, dnsv2.TypeA, "192.0.2.1"))
			},
			want: []string{`PUT /domains/1/records/21 {"ttl":600,"value":[{"enabled":true,"value":"192.0.2.1"}]}`},
		},
		{
			name:     "delete",
			existing: `[` + wwwA + `]`,
			desired:  apexNS,
			want:     []string{`DELETE /domains/1/records/21`},
		},
		{
			name:     "apex NS equal to Constellix's nameservers",
			existing: `[]`,
			desired:  apexNS,
			want:     nil,
		},
		{
			name:     "foreign apex NS",
			existing: `[]`,
			desired: func(dc *models.DomainConfig) models.Records {
				return append(apexNS(dc), dc.MustNewRecordConfig("@", 300, dnsv2.TypeNS, "ns1.other-provider.example."))
			},
			want: []string{`POST /domains/1/records {"name":"","type":"NS","ttl":300,"value":[{"enabled":true,"host":"ns1.other-provider.example."}]}`},
		},
		{
			name:     "stored apex NS repeating a Constellix nameserver, plus a foreign one",
			existing: `[` + apexNSOwn + `]`,
			desired: func(dc *models.DomainConfig) models.Records {
				return append(apexNS(dc), dc.MustNewRecordConfig("@", 300, dnsv2.TypeNS, "ns1.other-provider.example."))
			},
			want: []string{`PUT /domains/1/records/31 {"ttl":300,"value":[{"enabled":true,"host":"ns1.other-provider.example."}]}`},
		},
		{
			name:     "stored apex NS repeating a Constellix nameserver, nothing configured",
			existing: `[` + apexNSOwn + `]`,
			desired:  apexNS,
			want:     []string{`DELETE /domains/1/records/31`},
		},
		{
			name:     "TTL above the maximum",
			existing: `[]`,
			desired: func(dc *models.DomainConfig) models.Records {
				return append(apexNS(dc), dc.MustNewRecordConfig("www", 4294967295, dnsv2.TypeA, "192.0.2.1"))
			},
			want: []string{`POST /domains/1/records {"name":"www","type":"A","ttl":2147483647,"value":[{"enabled":true,"value":"192.0.2.1"}]}`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newFakeAPI(t)
			api.records["1"] = tt.existing
			got := corrections(t, api, tt.desired)
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("writes =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

// TestGetZoneRecordsCorrectionsConverge checks that the zone as the API keeps
// it after a push produces no further correction.
func TestGetZoneRecordsCorrectionsConverge(t *testing.T) {
	tests := []struct {
		name    string
		stored  string
		desired func(dc *models.DomainConfig) models.Records
	}{
		{
			name:    "the redundant apex NS set was deleted",
			stored:  `[]`,
			desired: apexNS,
		},
		{
			name:   "a TTL above the maximum was stored as the maximum",
			stored: `[{"id": 21, "name": "www", "type": "A", "ttl": 2147483647, "value": [{"value": "192.0.2.1", "enabled": true}]}]`,
			desired: func(dc *models.DomainConfig) models.Records {
				return append(apexNS(dc), dc.MustNewRecordConfig("www", 4294967295, dnsv2.TypeA, "192.0.2.1"))
			},
		},
		{
			name:   "a foreign apex NS",
			stored: `[{"id": 31, "name": "", "type": "NS", "ttl": 300, "value": [{"host": "ns1.other-provider.example.", "enabled": true}]}]`,
			desired: func(dc *models.DomainConfig) models.Records {
				return append(apexNS(dc), dc.MustNewRecordConfig("@", 300, dnsv2.TypeNS, "ns1.other-provider.example."))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newFakeAPI(t)
			api.records["1"] = tt.stored
			if got := corrections(t, api, tt.desired); len(got) != 0 {
				t.Errorf("writes = %v, want none", got)
			}
		})
	}
}

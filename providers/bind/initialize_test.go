package bind

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func TestRegisteredDefinition(t *testing.T) {
	def, ok := providers.GetDefinition("BIND")
	if !ok || def.Kind != providers.KindDNS || !def.CanGetZones || !def.DocCreateDomains {
		t.Fatalf("BIND definition = %+v", def)
	}
	if !reflect.DeepEqual(def.DerivedFeatures, def.Features) {
		t.Fatal("migration changed legacy capabilities or documentation notes")
	}
	if errors := providers.AuditRecords("BIND", nil); len(errors) != 0 {
		t.Fatal(errors)
	}
	directory := filepath.Join(t.TempDir(), "zones")
	if err := def.PostWrite(map[string]string{"directory": directory}); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(directory); err != nil || !st.IsDir() {
		t.Fatalf("PostWrite did not create directory: %v", err)
	}
}

func TestInitializeConfiguration(t *testing.T) {
	instance, err := providers.CreateDNSProvider("BIND", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := instance.(*bindProvider)
	if p.directory != defaultZonesDir || p.filenameformat != "%c.zone" {
		t.Fatal("BIND defaults changed")
	}
	directory := filepath.Join(t.TempDir(), "uncreated")
	instance, err = providers.CreateDNSProvider("BIND", map[string]string{"directory": directory, "filenameformat": "%c.test"}, json.RawMessage(`{"default_ns":["ns1.example.com."],"default_soa":{"master":"ns1.example.com.","serial":123}}`))
	if err != nil {
		t.Fatal(err)
	}
	q := instance.(*bindProvider)
	if p == q || q.directory != directory || q.filenameformat != "%c.test" || q.DefaultSoa.Serial != 123 {
		t.Fatal("BIND instance did not retain configuration and metadata")
	}
	if ns, err := q.GetNameservers("example.com"); err != nil || len(ns) != 1 || ns[0].Name != "ns1.example.com" {
		t.Fatalf("BIND nameservers: %v, %v", ns, err)
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatal("BIND initialization created the directory before PostWrite")
	}
	for _, meta := range []json.RawMessage{json.RawMessage(`{broken`), json.RawMessage(`{"default_ns":[""]}`), json.RawMessage(`{"default_ns":["ns1.example.com"]}`)} {
		if p, err := providers.CreateDNSProvider("BIND", nil, meta); p != nil || err == nil {
			t.Fatalf("invalid metadata accepted: %v, %v", p, err)
		}
	}
}

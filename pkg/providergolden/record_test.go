package providergolden

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

type fakeNative struct {
	Name string `json:"name"`
}

func TestRecorderWritesIndexedInputAndOutput(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	records := models.Records{
		dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1"),
		dc.MustNewRecordConfig("www", 300, "AAAA", "2001:db8::1"),
	}
	recorder := NewRecorder()
	observer := recorder.ForDomain(dc.Name)

	native := fakeNative{Name: "www"}
	toRC := observer.BeginToRC("toRC", native)
	observer.EndToRC("toRC", toRC, native, records, nil)
	toNative := observer.BeginToNative("toNative", records)
	observer.EndToNative("toNative", toNative, records, native, nil)

	dir := t.TempDir()
	written, err := recorder.WriteTo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 5 {
		t.Fatalf("wrote %d files, want 5: %v", len(written), written)
	}

	metadata, err := readMetadata(dir)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Version != 1 || len(metadata.Recordings) != 1 {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	got := metadata.Recordings[0]
	if got.Domain != "example.com" || strings.Join(got.ToRC, ",") != "toRC" || strings.Join(got.ToNative, ",") != "toNative" {
		t.Fatalf("unexpected recording descriptor: %#v", got)
	}

	assertFileContains(t, filepath.Join(dir, toRCOutputFile("toRC", dc.Name)), "1\twww 300 IN A 192.0.2.1\n1\twww 300 IN AAAA 2001:db8::1\n")
	assertFileContains(t, filepath.Join(dir, toNativeInputFile("toNative", dc.Name)), "1\twww 300 IN A 192.0.2.1\n1\twww 300 IN AAAA 2001:db8::1\n")

	var input []indexedValue[fakeNative]
	data, err := os.ReadFile(filepath.Join(dir, toRCInputFile("toRC", dc.Name)))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	if len(input) != 1 || input[0].Index != 1 || input[0].Value != native {
		t.Fatalf("unexpected indexed input: %#v", input)
	}
}

func TestRecorderRejectsMutation(t *testing.T) {
	recorder := NewRecorder()
	observer := recorder.ForDomain("example.com")
	native := &fakeNative{Name: "before"}
	snapshot := observer.BeginToRC("toRC", native)
	native.Name = "after"
	observer.EndToRC("toRC", snapshot, native, nil, nil)
	if _, err := recorder.WriteTo(t.TempDir()); err == nil || !strings.Contains(err.Error(), "mutated its native input") {
		t.Fatalf("WriteTo() error = %v, want mutation error", err)
	}
}

func TestRecorderMergesDomains(t *testing.T) {
	dir := t.TempDir()
	metadata := recordingMetadata{Version: 1, Recordings: []recordingDescriptor{{Domain: "old.example", ToRC: []string{"old"}}}}
	data, _ := json.Marshal(metadata)
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	recorder := NewRecorder()
	observer := recorder.ForDomain("new.example")
	native := fakeNative{Name: "new"}
	snapshot := observer.BeginToRC("new", native)
	observer.EndToRC("new", snapshot, native, nil, nil)
	if _, err := recorder.WriteTo(dir); err != nil {
		t.Fatal(err)
	}
	got, err := readMetadata(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Recordings) != 2 || got.Recordings[0].Domain != "new.example" || got.Recordings[1].Domain != "old.example" {
		t.Fatalf("merged metadata = %#v", got.Recordings)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

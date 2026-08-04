package providergolden

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

type fakeNative struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type fakeProvider struct {
	corrections []*models.Correction
	count       int
	err         error
	calls       int
}

func (p *fakeProvider) GetNameservers(string) ([]*models.Nameserver, error) { return nil, nil }

func (p *fakeProvider) GetZoneRecords(*models.DomainConfig) (models.Records, error) {
	return nil, nil
}

func (p *fakeProvider) GetZoneRecordsCorrections(*models.DomainConfig, models.Records) ([]*models.Correction, int, error) {
	p.calls++
	return p.corrections, p.count, p.err
}

func TestRecorderWritesTheRecordsItObserved(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	desired := models.Records{
		dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1"),
		dc.MustNewRecordConfig("@", 3600, "MX", 10, "mail.example.com."),
		dc.MustNewRecordConfig("mail", 300, "A", "192.0.2.2"),
		dc.MustNewRecordConfig("@", 600, "TXT", "hello world"),
	}

	rec := NewRecorder()
	rec.Observe(desired, nil)

	dir := t.TempDir()
	written, err := rec.WriteTo(dir, "example")
	if err != nil {
		t.Fatalf("WriteTo() error: %v", err)
	}
	if want := []string{filepath.Join(dir, "example.records")}; len(written) != 1 || written[0] != want[0] {
		t.Fatalf("WriteTo() wrote %v, want %v", written, want)
	}

	got, err := os.ReadFile(written[0])
	if err != nil {
		t.Fatal(err)
	}
	want := "@ 3600 IN MX 10 mail.example.com.\n" +
		"@ 600 IN TXT \"hello world\"\n" +
		"mail 300 IN A 192.0.2.2\n" +
		"www 300 IN A 192.0.2.1\n"
	if string(got) != want {
		t.Errorf("example.records =\n%s\nwant:\n%s", got, want)
	}

	recs, err := parseRecords(dc, string(got))
	if err != nil {
		t.Fatalf("parseRecords() rejected the recorded file: %v", err)
	}
	if len(recs) != len(desired) {
		t.Errorf("parseRecords() returned %d records, want %d", len(recs), len(desired))
	}
}

func TestRecorderWritesTheNativeRecordsBehindTheConvertedOnes(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	withOriginal := dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1")
	withOriginal.Original = fakeNative{Name: "www", Type: "A"}
	withoutOriginal := dc.MustNewRecordConfig("mail", 300, "A", "192.0.2.2")

	rec := NewRecorder()
	rec.Observe(nil, models.Records{withOriginal, withoutOriginal})

	dir := t.TempDir()
	written, err := rec.WriteTo(dir, "example")
	if err != nil {
		t.Fatalf("WriteTo() error: %v", err)
	}
	if want := []string{filepath.Join(dir, "example.json")}; len(written) != 1 || written[0] != want[0] {
		t.Fatalf("WriteTo() wrote %v, want %v", written, want)
	}

	data, err := os.ReadFile(written[0])
	if err != nil {
		t.Fatal(err)
	}
	var natives []fakeNative
	if err := json.Unmarshal(data, &natives); err != nil {
		t.Fatalf("example.json: %v", err)
	}
	if len(natives) != 1 {
		t.Fatalf("example.json holds %d natives, want 1", len(natives))
	}
	if want := (fakeNative{Name: "www", Type: "A"}); natives[0] != want {
		t.Errorf("example.json holds %+v, want %+v", natives[0], want)
	}
}

func TestRecorderDiscardsDuplicates(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1")
	rc.Original = fakeNative{Name: "www", Type: "A"}

	rec := NewRecorder()
	for range 3 {
		rec.Observe(models.Records{rc}, models.Records{rc})
	}

	dir := t.TempDir()
	if _, err := rec.WriteTo(dir, "example"); err != nil {
		t.Fatalf("WriteTo() error: %v", err)
	}

	records, err := os.ReadFile(filepath.Join(dir, "example.records"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "www 300 IN A 192.0.2.1\n"; string(records) != want {
		t.Errorf("example.records = %q, want %q", records, want)
	}

	data, err := os.ReadFile(filepath.Join(dir, "example.json"))
	if err != nil {
		t.Fatal(err)
	}
	var natives []fakeNative
	if err := json.Unmarshal(data, &natives); err != nil {
		t.Fatal(err)
	}
	if len(natives) != 1 {
		t.Errorf("example.json holds %d natives, want 1", len(natives))
	}
}

func TestRecorderReportsANativeItCannotMarshal(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1")
	rc.Original = make(chan int)

	rec := NewRecorder()
	rec.Observe(nil, models.Records{rc})

	written, err := rec.WriteTo(t.TempDir(), "example")
	if err == nil {
		t.Error("WriteTo() returned no error for a native that cannot be marshalled")
	}
	if len(written) != 0 {
		t.Errorf("WriteTo() wrote %v, want nothing", written)
	}
}

func TestRecorderWritesNothingWhenItObservedNothing(t *testing.T) {
	dir := t.TempDir()

	written, err := NewRecorder().WriteTo(dir, "example")
	if err != nil {
		t.Fatalf("WriteTo() error: %v", err)
	}
	if len(written) != 0 {
		t.Errorf("WriteTo() wrote %v, want nothing", written)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("WriteTo() created %d files, want 0", len(entries))
	}
}

func TestRecordObservesTheConversionsAndReturnsWhatTheProviderReturned(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	dc.Records = models.Records{dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1")}

	existing := dc.MustNewRecordConfig("mail", 300, "A", "192.0.2.2")
	existing.Original = fakeNative{Name: "mail", Type: "A"}

	wantErr := errors.New("provider failed")
	fake := &fakeProvider{
		corrections: []*models.Correction{{Msg: "a correction"}},
		count:       7,
		err:         wantErr,
	}

	rec := NewRecorder()
	corrections, count, err := Record(fake, rec).GetZoneRecordsCorrections(dc, models.Records{existing})

	if fake.calls != 1 {
		t.Errorf("provider was called %d times, want 1", fake.calls)
	}
	if len(corrections) != 1 || corrections[0].Msg != "a correction" {
		t.Errorf("corrections = %v, want the provider's own", corrections)
	}
	if count != 7 {
		t.Errorf("count = %d, want 7", count)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}

	dir := t.TempDir()
	written, err := rec.WriteTo(dir, "example")
	if err != nil {
		t.Fatalf("WriteTo() error: %v", err)
	}
	want := []string{filepath.Join(dir, "example.records"), filepath.Join(dir, "example.json")}
	if len(written) != len(want) || written[0] != want[0] || written[1] != want[1] {
		t.Fatalf("WriteTo() wrote %v, want %v", written, want)
	}

	records, err := os.ReadFile(written[0])
	if err != nil {
		t.Fatal(err)
	}
	if want := "www 300 IN A 192.0.2.1\n"; string(records) != want {
		t.Errorf("example.records = %q, want %q", records, want)
	}
}

func TestProviderNameIsThePackageTheProviderIsIn(t *testing.T) {
	name, err := ProviderName(&fakeProvider{})
	if err != nil {
		t.Fatalf("ProviderName() error: %v", err)
	}
	if name != "providergolden" {
		t.Errorf("ProviderName() = %q, want %q", name, "providergolden")
	}
}

func TestARecordingIsNamedTheWayTheChecksReadIt(t *testing.T) {
	name, err := ProviderName(&fakeProvider{})
	if err != nil {
		t.Fatalf("ProviderName() error: %v", err)
	}

	dc := models.MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1")
	rc.Original = fakeNative{Name: "www", Type: "A"}

	rec := NewRecorder()
	rec.Observe(models.Records{rc}, models.Records{rc})

	packageDir := filepath.Join(t.TempDir(), name)
	if _, err := rec.WriteTo(filepath.Join(packageDir, testdataDir), name); err != nil {
		t.Fatalf("WriteTo() error: %v", err)
	}

	t.Chdir(packageDir)
	for _, ext := range []string{nativesExt, recordsExt} {
		input, err := inputFile(ext)
		if err != nil {
			t.Fatalf("inputFile(%q) error: %v", ext, err)
		}
		if _, ok, err := loadInput(testdataDir, input); err != nil {
			t.Fatalf("loadInput(%q) error: %v", input, err)
		} else if !ok {
			t.Errorf("the recording is not readable as %s", filepath.Join(testdataDir, input))
		}
	}
}

func TestTestdataDirIsTheProvidersOwnTestdataDirectory(t *testing.T) {
	dir, err := TestdataDir(&fakeProvider{})
	if err != nil {
		t.Fatalf("TestdataDir() error: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("TestdataDir() = %q, want an absolute path", dir)
	}
	if want := filepath.Join("providers", "providergolden", testdataDir); !strings.HasSuffix(dir, want) {
		t.Errorf("TestdataDir() = %q, want a path ending in %q", dir, want)
	}
}

func TestResolveDirIsRelativeToTheModuleRoot(t *testing.T) {
	rel := filepath.Join("providers", "bind", testdataDir)

	dir, err := ResolveDir(rel)
	if err != nil {
		t.Fatalf("ResolveDir(%q) error: %v", rel, err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if dir == filepath.Join(wd, rel) {
		t.Errorf("ResolveDir(%q) = %q, want a path under the module root, not the working directory", rel, dir)
	}

	root := strings.TrimSuffix(dir, string(filepath.Separator)+rel)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Errorf("ResolveDir(%q) = %q, want a path under the directory holding go.mod: %v", rel, dir, err)
	}
}

func TestResolveDirKeepsAnAbsolutePath(t *testing.T) {
	want := t.TempDir()

	dir, err := ResolveDir(want)
	if err != nil {
		t.Fatalf("ResolveDir(%q) error: %v", want, err)
	}
	if dir != want {
		t.Errorf("ResolveDir(%q) = %q, want %q", want, dir, want)
	}
}

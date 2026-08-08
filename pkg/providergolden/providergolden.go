// Package providergolden replays recorded provider data through a provider's
// record conversion functions and compares the result with a golden file.
//
// A provider is enrolled by adding one small test per conversion function. The
// test names the recorded data and adapts the provider's function to a uniform
// signature:
//
//	func TestToRecordConfig(t *testing.T) {
//		providergolden.CheckToRC(t, "netlify_torecordconfig",
//			func(dc *models.DomainConfig, n dnsRecord) ([]*models.RecordConfig, error) {
//				rc, err := toRecordConfig(dc, &n)
//				return []*models.RecordConfig{rc}, err
//			})
//	}
//
// When both conversion directions use the same native type, CheckRoundTrip
// verifies that converting a recorded RecordConfig to the native type and back
// preserves its StringWithMeta representation.
//
// The data lives in the provider's test_data directory. A recording covers the
// whole provider, so the input files are named after it, and the golden file is
// named after the test that produced it:
//
//	test_data/<provider>.meta.json  recording metadata, including the zone
//	test_data/<provider>.json       native records, as the provider's API returns them
//	test_data/<provider>.records    DNS records, in the golden line format below
//	test_data/<name>.golden         the expected output
//
// <provider> is the directory the test is running in, which is the same name
// the integration tests record under, so a recording needs no renaming and one
// recording feeds every test the provider has.
//
// CheckToRC reads the .json file, CheckToNative reads the .records file, and
// both write the .golden file. A provider with no recorded data is skipped, so
// providers can be enrolled one at a time.
//
// "go test <package> -update" rewrites the golden files from the recorded
// inputs. It never contacts a provider's API and never rewrites an input file,
// so gathering data from a live account stays a separate, explicit step.
//
// That step is Recorder, which collects both kinds of input from a provider as
// it is used. The integration tests wrap their provider in one when they are
// given "-record", and write what it collected to TestdataDir.
//
// A golden line is the record's label, TTL, class, type and RDATA, followed by
// the metadata when the record has any:
//
//	www 300 IN A 192.0.2.1
//	@ 3600 IN MX 10 mail.example.com.
//	fwd 0 IN URL https://example.net/landing ; includePath="no" type="temporary" wildcard="no"
package providergolden

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/google/go-cmp/cmp"
)

var update = flag.Bool("update", false, "rewrite the provider conversion golden files")

const (
	testDataDir = "test_data"

	// Extensions of the two kinds of recorded input, written by Recorder and
	// read by CheckToRC and CheckToNative.
	metadataExt = ".meta.json"
	nativesExt  = ".json"
	recordsExt  = ".records"
)

type recordingMetadata struct {
	Domain string `json:"domain"`
}

// CheckToRC replays the native records recorded in test_data/<provider>.json
// through convert and compares the records it returns with
// test_data/<name>.golden.
func CheckToRC[N any](t *testing.T, name string, convert func(dc *models.DomainConfig, native N) ([]*models.RecordConfig, error)) {
	t.Helper()

	input, err := inputFile(nativesExt)
	if err != nil {
		t.Fatal(err)
	}

	data, ok, err := loadInput(testDataDir, input)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Skipf("%s has no recorded data yet", filepath.Join(testDataDir, input))
	}

	var natives []N
	if err := json.Unmarshal(data, &natives); err != nil {
		t.Fatalf("%s: %v", input, err)
	}

	dc := models.MustNewDomainConfig(RecordedDomain(t))
	var b strings.Builder
	for i, native := range natives {
		recs, err := convert(dc, native)
		if err != nil {
			t.Fatalf("%s: record %d: %v", input, i, err)
		}
		for _, rc := range recs {
			if rc == nil {
				continue
			}
			b.WriteString(rc.StringWithMeta())
			b.WriteByte('\n')
		}
	}

	report(t, testDataDir, name, []byte(b.String()))
}

// CheckToNative replays the records recorded in test_data/<provider>.records
// through convert and compares the native records it returns with
// test_data/<name>.golden.
func CheckToNative[N any](t *testing.T, name string, convert func(rc *models.RecordConfig) (N, error)) {
	t.Helper()

	input, err := inputFile(recordsExt)
	if err != nil {
		t.Fatal(err)
	}

	data, ok, err := loadInput(testDataDir, input)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Skipf("%s has no recorded data yet", filepath.Join(testDataDir, input))
	}

	recs, err := parseRecords(models.MustNewDomainConfig(RecordedDomain(t)), string(data))
	if err != nil {
		t.Fatalf("%s: %v", input, err)
	}

	natives := make([]N, 0, len(recs))
	for i, rc := range recs {
		native, err := convert(rc)
		if err != nil {
			t.Fatalf("%s: record %d: %v", input, i, err)
		}
		natives = append(natives, native)
	}

	got, err := json.MarshalIndent(natives, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	report(t, testDataDir, name, append(got, '\n'))
}

// CheckRoundTrip replays the records in test_data/<provider>.records through
// both provider conversion functions and verifies that their DNSControl
// representation is unchanged.
func CheckRoundTrip[N any](t *testing.T,
	toNative func(rc *models.RecordConfig) (N, error),
	toRC func(dc *models.DomainConfig, native N) ([]*models.RecordConfig, error),
) {
	t.Helper()

	input, err := inputFile(recordsExt)
	if err != nil {
		t.Fatal(err)
	}

	data, ok, err := loadInput(testDataDir, input)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Skipf("%s has no recorded data yet", filepath.Join(testDataDir, input))
	}

	dc := models.MustNewDomainConfig(RecordedDomain(t))
	recs, err := parseRecords(dc, string(data))
	if err != nil {
		t.Fatalf("%s: %v", input, err)
	}

	for i, before := range recs {
		native, err := toNative(before)
		if err != nil {
			t.Fatalf("%s: record %d: convert to native: %v", input, i, err)
		}
		after, err := toRC(dc, native)
		if err != nil {
			t.Fatalf("%s: record %d: convert back to RecordConfig: %v", input, i, err)
		}

		after = nonNilRecords(after)
		if len(after) != 1 {
			t.Errorf("%s: record %d: round trip returned %d records, want 1", input, i, len(after))
			continue
		}
		if want, got := before.StringWithMeta(), after[0].StringWithMeta(); got != want {
			t.Errorf("%s: record %d changed in round trip\nwant: %s\n got: %s", input, i, want, got)
		}
	}
}

// RecordedDomain returns the zone captured alongside the provider data. The
// environment is deliberately not consulted: replaying a recording must not
// depend on the credentials or integration-test zone of the person running it.
func RecordedDomain(t *testing.T) string {
	t.Helper()

	input, err := inputFile(metadataExt)
	if err != nil {
		t.Fatal(err)
	}
	data, ok, err := loadInput(testDataDir, input)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("%s is missing; record the provider again with -record", filepath.Join(testDataDir, input))
	}

	var metadata recordingMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("%s: %v", input, err)
	}
	if metadata.Domain == "" {
		t.Fatalf("%s: domain is empty", input)
	}
	return metadata.Domain
}

func nonNilRecords(recs []*models.RecordConfig) []*models.RecordConfig {
	n := 0
	for _, rc := range recs {
		if rc != nil {
			recs[n] = rc
			n++
		}
	}
	return recs[:n]
}

// inputFile returns the recorded input file with extension ext that belongs to
// the provider under test: the directory the test is running in is the
// provider's package directory, and a recording is named after it.
func inputFile(ext string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Base(wd) + ext, nil
}

// loadInput reads a recorded input file. ok is false when the file does not
// exist, which means the provider has not been enrolled yet.
func loadInput(dir, filename string) (data []byte, ok bool, err error) {
	data, err = os.ReadFile(filepath.Join(dir, filename))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

// report compares got with the golden file, or rewrites the golden file when
// -update was given.
func report(t *testing.T, dir, name string, got []byte) {
	t.Helper()

	skip, diff, err := compareGolden(dir, name, got, *update)
	switch {
	case err != nil:
		t.Fatal(err)
	case skip != "":
		t.Skip(skip)
	case diff != "":
		t.Errorf("%s.golden does not match the conversion (-want +got):\n%s", name, diff)
	}
}

// compareGolden compares got with <dir>/<name>.golden, or writes got to it when
// update is true. It returns a reason to skip when the golden file does not
// exist, and a diff when the contents differ.
func compareGolden(dir, name string, got []byte, update bool) (skip, diff string, err error) {
	path := filepath.Join(dir, name+".golden")

	if update {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", "", err
		}
		return "", "", os.WriteFile(path, got, 0o644)
	}

	want, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Sprintf("%s does not exist: run \"go test . -update\" to record it", path), "", nil
	}
	if err != nil {
		return "", "", err
	}

	return "", cmp.Diff(strings.Split(string(want), "\n"), strings.Split(string(got), "\n")), nil
}

// parseRecords parses the golden line format. Blank lines are ignored.
func parseRecords(dc *models.DomainConfig, text string) ([]*models.RecordConfig, error) {
	var recs []*models.RecordConfig
	for i, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		rc, err := parseRecord(dc, line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		recs = append(recs, rc)
	}
	return recs, nil
}

func parseRecord(dc *models.DomainConfig, line string) (*models.RecordConfig, error) {
	record, metatext := cutMetadata(line)

	fields := strings.SplitN(record, " ", 5)
	if len(fields) != 5 {
		return nil, fmt.Errorf("%q: expected \"label ttl IN type rdata\"", line)
	}
	name, ttltext, class, rtype, rdata := fields[0], fields[1], fields[2], fields[3], fields[4]

	ttl, err := strconv.ParseUint(ttltext, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", line, err)
	}
	if class != "IN" {
		return nil, fmt.Errorf("%q: expected class \"IN\"", line)
	}

	rc, err := dc.NewRecordConfigParse(name, uint32(ttl), rtype, rdata)
	if err != nil {
		return nil, err
	}

	metadata, err := parseMetadata(metatext)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", line, err)
	}
	if len(metadata) != 0 {
		rc.Metadata = metadata
	}

	return rc, nil
}

// cutMetadata splits a line at the first semicolon that is not inside a quoted
// string.
func cutMetadata(line string) (record, metadata string) {
	quoted := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '"':
			quoted = !quoted
		case ';':
			if !quoted {
				return strings.TrimRight(line[:i], " "), line[i+1:]
			}
		}
	}
	return line, ""
}

// parseMetadata parses a sequence of key="value" pairs.
func parseMetadata(s string) (map[string]string, error) {
	metadata := map[string]string{}
	s = strings.TrimLeft(s, " ")
	for s != "" {
		key, rest, ok := strings.Cut(s, "=")
		if !ok {
			return nil, fmt.Errorf("metadata %q: expected key=\"value\"", s)
		}
		value, rest, err := cutQuoted(rest)
		if err != nil {
			return nil, err
		}
		metadata[key] = value
		s = strings.TrimLeft(rest, " ")
	}
	return metadata, nil
}

// cutQuoted removes a Go-quoted string from the front of s.
func cutQuoted(s string) (value, rest string, err error) {
	if !strings.HasPrefix(s, `"`) {
		return "", "", fmt.Errorf("metadata %q: expected a quoted value", s)
	}
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '"':
			value, err := strconv.Unquote(s[:i+1])
			if err != nil {
				return "", "", fmt.Errorf("metadata %q: %w", s, err)
			}
			return value, s[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("metadata %q: unterminated quoted value", s)
}

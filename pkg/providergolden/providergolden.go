// Package providergolden replays recorded provider data through a provider's
// record conversion functions and compares the result with a golden file.
//
// A provider is enrolled by adding one small test per conversion function. The
// test names the recorded data and adapts the provider's function to a uniform
// signature:
//
//	func TestToRecordConfig(t *testing.T) {
//		providergolden.CheckToRC(t, "websupport_torecordconfig", "example.com",
//			func(dc *models.DomainConfig, n nativeRecord) ([]*models.RecordConfig, error) {
//				rc, err := toRecordConfig(dc, n)
//				return []*models.RecordConfig{rc}, err
//			})
//	}
//
// The data lives in the provider's testdata directory and is named after the
// provider and the function under test:
//
//	testdata/<name>.json     native records, as the provider's API returns them
//	testdata/<name>.records  DNS records, in the golden line format below
//	testdata/<name>.golden   the expected output
//
// CheckToRC reads the .json file, CheckToNative reads the .records file, and
// both write the .golden file. A provider with no recorded data is skipped, so
// providers can be enrolled one at a time.
//
// "go test <package> -update" rewrites the golden files from the recorded
// inputs. It never contacts a provider's API and never rewrites an input file,
// so gathering data from a live account stays a separate, explicit step.
//
// A golden line is the record's label, TTL, class, type and RDATA, with the TTL
// omitted when it is zero, followed by the metadata when the record has any:
//
//	www 300 IN A 192.0.2.1
//	@ IN MX 10 mail.example.com.
//	fwd IN URL https://example.net/landing ; includePath="no" type="temporary" wildcard="no"
package providergolden

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/google/go-cmp/cmp"
)

var update = flag.Bool("update", false, "rewrite the provider conversion golden files")

const testdataDir = "testdata"

// CheckToRC replays the native records recorded in testdata/<name>.json through
// convert and compares the records it returns with testdata/<name>.golden.
func CheckToRC[N any](t *testing.T, name, domain string, convert func(dc *models.DomainConfig, native N) ([]*models.RecordConfig, error)) {
	t.Helper()

	data, ok, err := loadInput(testdataDir, name+".json")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Skipf("%s has no recorded data yet", name)
	}

	var natives []N
	if err := json.Unmarshal(data, &natives); err != nil {
		t.Fatalf("%s.json: %v", name, err)
	}

	dc := models.MustNewDomainConfig(domain)
	var b strings.Builder
	for i, native := range natives {
		recs, err := convert(dc, native)
		if err != nil {
			t.Fatalf("%s.json: record %d: %v", name, i, err)
		}
		for _, rc := range recs {
			if rc == nil {
				continue
			}
			b.WriteString(formatRecord(rc))
			b.WriteByte('\n')
		}
	}

	report(t, testdataDir, name, []byte(b.String()))
}

// CheckToNative replays the records recorded in testdata/<name>.records through
// convert and compares the native records it returns with testdata/<name>.golden.
func CheckToNative[N any](t *testing.T, name, domain string, convert func(rc *models.RecordConfig) (N, error)) {
	t.Helper()

	data, ok, err := loadInput(testdataDir, name+".records")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Skipf("%s has no recorded data yet", name)
	}

	recs, err := parseRecords(models.MustNewDomainConfig(domain), string(data))
	if err != nil {
		t.Fatalf("%s.records: %v", name, err)
	}

	natives := make([]N, 0, len(recs))
	for i, rc := range recs {
		native, err := convert(rc)
		if err != nil {
			t.Fatalf("%s.records: record %d: %v", name, i, err)
		}
		natives = append(natives, native)
	}

	got, err := json.MarshalIndent(natives, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	report(t, testdataDir, name, append(got, '\n'))
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

// formatRecord renders rc as one line of a golden file.
func formatRecord(rc *models.RecordConfig) string {
	var b strings.Builder

	b.WriteString(rc.Name)
	b.WriteByte(' ')
	if rc.TTL != 0 {
		b.WriteString(strconv.FormatUint(uint64(rc.TTL), 10))
		b.WriteByte(' ')
	}
	b.WriteString("IN ")
	b.WriteString(rc.Type)
	b.WriteByte(' ')
	b.WriteString(rc.GetRDATA().String())

	if len(rc.Metadata) != 0 {
		b.WriteString(" ;")
		for _, k := range slices.Sorted(maps.Keys(rc.Metadata)) {
			fmt.Fprintf(&b, " %s=%s", k, strconv.Quote(rc.Metadata[k]))
		}
	}

	return b.String()
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
	line, metatext := cutMetadata(line)

	name, rest, ok := strings.Cut(line, " ")
	if !ok {
		return nil, fmt.Errorf("%q: expected \"label [ttl] IN type rdata\"", line)
	}

	var ttl uint64
	if head, tail, ok := strings.Cut(rest, " "); ok {
		if n, err := strconv.ParseUint(head, 10, 32); err == nil {
			ttl, rest = n, tail
		}
	}

	class, rest, ok := strings.Cut(rest, " ")
	if !ok || class != "IN" {
		return nil, fmt.Errorf("%q: expected class \"IN\"", line)
	}

	rtype, rdata, ok := strings.Cut(rest, " ")
	if !ok {
		return nil, fmt.Errorf("%q: expected rdata after the type", line)
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

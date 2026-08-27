// Package providergolden records and replays exact provider conversion calls.
package providergolden

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/google/go-cmp/cmp"
)

var update = flag.Bool("update", false, "rewrite provider conversion expected-output files")

const testDataDir = "test_data"

var safeFilenamePart = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func conversionFilename(kind, function, domain, ext string) string {
	if !safeFilenamePart.MatchString(function) || !safeFilenamePart.MatchString(domain) {
		panic(fmt.Sprintf("unsafe conversion filename: function=%q domain=%q", function, domain))
	}
	return fmt.Sprintf("%s_%s_%s.%s", kind, function, domain, ext)
}

func toRCInputFile(function, domain string) string {
	return conversionFilename("recorded_torc_input", function, domain, "json")
}

func toRCOutputFile(function, domain string) string {
	return conversionFilename("expected_torc_output", function, domain, "records")
}

func toNativeInputFile(function, domain string) string {
	return conversionFilename("recorded_tonative_input", function, domain, "records")
}

func toNativeOutputFile(function, domain string) string {
	return conversionFilename("expected_tonative_output", function, domain, "json")
}

type indexedValue[T any] struct {
	Index int `json:"index"`
	Value T   `json:"value"`
}

// CheckToRC replays every domain recorded for function.
func CheckToRC[N any, R ~[]*models.RecordConfig](t *testing.T, function string, convert func(*models.DomainConfig, N) (R, error)) {
	t.Helper()
	metadata := loadMetadataForTest(t)
	found := false
	for _, recording := range metadata.Recordings {
		if !slices.Contains(recording.ToRC, function) {
			continue
		}
		found = true
		recording := recording
		t.Run(recording.Domain, func(t *testing.T) {
			input := toRCInputFile(function, recording.Domain)
			data := mustLoad(t, input)
			var natives []indexedValue[N]
			if err := json.Unmarshal(data, &natives); err != nil {
				t.Fatalf("%s: %v", input, err)
			}
			validateIndexes(t, input, indexesOf(natives))

			dc := models.MustNewDomainConfig(recording.Domain)
			var got strings.Builder
			for _, item := range natives {
				before := mustMarshal(t, item.Value)
				records, err := convert(dc, item.Value)
				if err != nil {
					t.Fatalf("%s: index %d: %v", input, item.Index, err)
				}
				after := mustMarshal(t, item.Value)
				if !bytes.Equal(before, after) {
					t.Errorf("%s: index %d: ToRC mutated its native input", input, item.Index)
				}
				for _, record := range recordStrings(models.Records(records)) {
					fmt.Fprintf(&got, "%d\t%s\n", item.Index, record)
				}
			}
			reportFile(t, toRCOutputFile(function, recording.Domain), []byte(got.String()))
		})
	}
	if !found {
		t.Skipf("no ToRC recording for %s", function)
	}
}

// CheckToNative replays every domain recorded for function. Records sharing an
// index are passed to convert as one conversion invocation.
func CheckToNative[N any](t *testing.T, function string, convert func(*models.DomainConfig, models.Records) (N, error)) {
	t.Helper()
	metadata := loadMetadataForTest(t)
	found := false
	for _, recording := range metadata.Recordings {
		if !slices.Contains(recording.ToNative, function) {
			continue
		}
		found = true
		recording := recording
		t.Run(recording.Domain, func(t *testing.T) {
			input := toNativeInputFile(function, recording.Domain)
			groups, err := parseIndexedRecords(models.MustNewDomainConfig(recording.Domain), string(mustLoad(t, input)))
			if err != nil {
				t.Fatalf("%s: %v", input, err)
			}
			outputs := make([]indexedValue[N], 0, len(groups))
			dc := models.MustNewDomainConfig(recording.Domain)
			for _, group := range groups {
				before := mustMarshal(t, group.Records)
				native, err := convert(dc, group.Records)
				if err != nil {
					t.Fatalf("%s: index %d: %v", input, group.Index, err)
				}
				after := mustMarshal(t, group.Records)
				if !bytes.Equal(before, after) {
					t.Errorf("%s: index %d: ToNative mutated its RecordConfig input", input, group.Index)
				}
				outputs = append(outputs, indexedValue[N]{Index: group.Index, Value: native})
			}
			data, err := json.MarshalIndent(outputs, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			reportFile(t, toNativeOutputFile(function, recording.Domain), append(data, '\n'))
		})
	}
	if !found {
		t.Skipf("no ToNative recording for %s", function)
	}
}

// CheckRoundTrip verifies each indexed ToNative input through both directions.
func CheckRoundTrip[N any, R ~[]*models.RecordConfig](t *testing.T, toNativeFunction string,
	toNative func(*models.DomainConfig, models.Records) (N, error),
	toRC func(*models.DomainConfig, N) (R, error),
) {
	t.Helper()
	metadata := loadMetadataForTest(t)
	for _, recording := range metadata.Recordings {
		if !slices.Contains(recording.ToNative, toNativeFunction) {
			continue
		}
		recording := recording
		t.Run(recording.Domain, func(t *testing.T) {
			dc := models.MustNewDomainConfig(recording.Domain)
			groups, err := parseIndexedRecords(dc, string(mustLoad(t, toNativeInputFile(toNativeFunction, recording.Domain))))
			if err != nil {
				t.Fatal(err)
			}
			for _, group := range groups {
				native, err := toNative(dc, group.Records)
				if err != nil {
					t.Fatalf("index %d: convert to native: %v", group.Index, err)
				}
				after, err := toRC(dc, native)
				if err != nil {
					t.Fatalf("index %d: convert to RecordConfig: %v", group.Index, err)
				}
				if diff := cmp.Diff(recordStrings(group.Records), recordStrings(models.Records(after))); diff != "" {
					t.Errorf("index %d changed in round trip (-before +after):\n%s", group.Index, diff)
				}
			}
		})
	}
}

type indexedRecordGroup struct {
	Index   int
	Records models.Records
}

func parseIndexedRecords(dc *models.DomainConfig, text string) ([]indexedRecordGroup, error) {
	var groups []indexedRecordGroup
	positions := map[int]int{}
	for lineNumber, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		prefix, record, ok := strings.Cut(line, "\t")
		if !ok {
			return nil, fmt.Errorf("line %d: expected index, tab, and record", lineNumber+1)
		}
		index64, err := strconv.ParseInt(prefix, 10, 32)
		if err != nil || index64 <= 0 {
			return nil, fmt.Errorf("line %d: invalid index %q", lineNumber+1, prefix)
		}
		index := int(index64)
		position, exists := positions[index]
		if !exists {
			if len(groups) != 0 && index <= groups[len(groups)-1].Index {
				return nil, fmt.Errorf("line %d: indexes must increase", lineNumber+1)
			}
			positions[index] = len(groups)
			groups = append(groups, indexedRecordGroup{Index: index})
			position = len(groups) - 1
		}
		rc, err := parseRecord(dc, record)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber+1, err)
		}
		groups[position].Records = append(groups[position].Records, rc)
	}
	return groups, nil
}

func indexesOf[T any](values []indexedValue[T]) []int {
	indexes := make([]int, len(values))
	for i := range values {
		indexes[i] = values[i].Index
	}
	return indexes
}

func validateIndexes(t *testing.T, filename string, indexes []int) {
	t.Helper()
	for i, index := range indexes {
		if index <= 0 || i != 0 && index <= indexes[i-1] {
			t.Fatalf("%s: indexes must be positive and increasing: %v", filename, indexes)
		}
	}
}

func loadMetadataForTest(t *testing.T) recordingMetadata {
	t.Helper()
	metadata, err := readMetadata(testDataDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Skipf("%s has no recorded data", filepath.Join(testDataDir, "meta.json"))
		}
		t.Fatal(err)
	}
	return metadata
}

func mustLoad(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(testDataDir, filename))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func reportFile(t *testing.T, filename string, got []byte) {
	t.Helper()
	path := filepath.Join(testDataDir, filename)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("%s does not exist: run go test . -update", path)
	}
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(strings.Split(string(want), "\n"), strings.Split(string(got), "\n")); diff != "" {
		t.Errorf("%s does not match (-want +got):\n%s", filename, diff)
	}
}

// // parseRecords parses unindexed record text. It remains useful for focused
// // parser tests and migration tooling.
// func parseRecords(dc *models.DomainConfig, text string) ([]*models.RecordConfig, error) {
// 	var recs []*models.RecordConfig
// 	for i, line := range strings.Split(text, "\n") {
// 		if strings.TrimSpace(line) == "" {
// 			continue
// 		}
// 		rc, err := parseRecord(dc, line)
// 		if err != nil {
// 			return nil, fmt.Errorf("line %d: %w", i+1, err)
// 		}
// 		recs = append(recs, rc)
// 	}
// 	return recs, nil
// }

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

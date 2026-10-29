package luadns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	api "github.com/luadns/luadns-go"
	"golang.org/x/time/rate"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
)

// recordedCall is one request received by fakeLuaDNS.
type recordedCall struct {
	Method string
	Path   string
	RRs    []string // sorted "name type content ttl"
}

// fakeLuaDNS answers the bulk endpoints and records every request.
// status, if set, picks the response code of the n-th (1-based) request.
type fakeLuaDNS struct {
	mu     sync.Mutex
	calls  []recordedCall
	status func(n int) int
}

func (f *fakeLuaDNS) recorded() []recordedCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// newFakeProvider returns a provider for zone example.com (ID 5) wired to a fake LuaDNS.
func newFakeProvider(t *testing.T, status func(n int) int) (*luadnsProvider, *fakeLuaDNS) {
	t.Helper()
	fake := &fakeLuaDNS{status: status}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body []api.RR
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("undecodable request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		rrs := make([]string, 0, len(body))
		for _, rr := range body {
			rrs = append(rrs, fmt.Sprintf("%s %s %s %d", rr.Name, rr.Type, rr.Content, rr.TTL))
		}
		slices.Sort(rrs)

		fake.mu.Lock()
		fake.calls = append(fake.calls, recordedCall{Method: r.Method, Path: r.URL.Path, RRs: rrs})
		n := len(fake.calls)
		fake.mu.Unlock()

		code := http.StatusOK
		if fake.status != nil {
			code = fake.status(n)
		}
		if code == http.StatusOK {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(code)
		if code == http.StatusOK {
			_, _ = w.Write([]byte("[]"))
		}
	}))
	t.Cleanup(srv.Close)

	return &luadnsProvider{
		provider:    api.NewClient("user@example.com", "key", api.SetBaseURL(srv.URL)),
		ctx:         context.Background(),
		rateLimiter: rate.NewLimiter(rate.Inf, 1),
		zones:       []*api.Zone{{ID: 5, Name: "example.com"}},
	}, fake
}

func newRC(t *testing.T, dc *models.DomainConfig, label, rtype, content string) *models.RecordConfig {
	t.Helper()
	var rc *models.RecordConfig
	var err error
	if rtype == "TXT" {
		rc, err = dc.NewRecordConfig(label, 300, rtype, content)
	} else {
		rc, err = dc.NewRecordConfigParse(label, 300, rtype, content)
	}
	if err != nil {
		t.Fatal(err)
	}
	return rc
}

func runAll(t *testing.T, corrections []*models.Correction) {
	t.Helper()
	for _, c := range corrections {
		if c.F == nil {
			continue
		}
		if err := c.F(); err != nil {
			t.Fatalf("%q: %v", c.Msg, err)
		}
	}
}

func sameCalls(got, want []recordedCall) bool {
	return slices.EqualFunc(got, want, func(a, b recordedCall) bool {
		return a.Method == b.Method && a.Path == b.Path && slices.Equal(a.RRs, b.RRs)
	})
}

// Only adjacent changes of the same kind share a request. The order diff2
// chose (here by label, with an A -> CNAME swap at the end) must reach the
// API unchanged, so the creates around the update and the delete stay apart.
func TestGetZoneRecordsCorrectionsBatchesAdjacentRuns(t *testing.T) {
	l, fake := newFakeProvider(t, nil)
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	existing := models.Records{
		newRC(t, dc, "c", "A", "3.3.3.3"),
		newRC(t, dc, "m", "A", "9.9.9.9"),
		newRC(t, dc, "x", "A", "8.8.8.8"),
	}
	dc.Records = models.Records{
		newRC(t, dc, "a", "A", "1.1.1.1"),
		newRC(t, dc, "b", "A", "2.2.2.2"),
		newRC(t, dc, "c", "A", "3.3.3.4"),
		newRC(t, dc, "c", "A", "3.3.3.5"),
		newRC(t, dc, "n", "A", "4.4.4.4"),
		newRC(t, dc, "o", "A", "5.5.5.5"),
		newRC(t, dc, "x", "CNAME", "target.example.com."),
	}

	corrections, _, err := l.GetZoneRecordsCorrections(dc, existing)
	if err != nil {
		t.Fatal(err)
	}
	runAll(t, corrections)

	want := []recordedCall{
		{"POST", "/zones/5/records/create_many", []string{
			"a.example.com. A 1.1.1.1 300",
			"b.example.com. A 2.2.2.2 300",
		}},
		{"PATCH", "/zones/5/records", []string{
			"c.example.com. A 3.3.3.4 300",
			"c.example.com. A 3.3.3.5 300",
		}},
		{"POST", "/zones/5/records/delete_many", []string{
			"m.example.com. A 9.9.9.9 300",
		}},
		{"POST", "/zones/5/records/create_many", []string{
			"n.example.com. A 4.4.4.4 300",
			"o.example.com. A 5.5.5.5 300",
		}},
		{"POST", "/zones/5/records/delete_many", []string{
			"x.example.com. A 8.8.8.8 300",
		}},
		{"POST", "/zones/5/records/create_many", []string{
			"x.example.com. CNAME target.example.com. 300",
		}},
	}
	if got := fake.recorded(); !sameCalls(got, want) {
		t.Errorf("requests mismatch\n got: %+v\nwant: %+v", got, want)
	}

	// Preview counts the lines of Msg, so batching must not drop any.
	changes, _, err := diff2.ByRecordSet(existing, dc, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantLines, gotLines := 0, 0
	for _, c := range changes {
		wantLines += len(strings.Split(c.MsgsJoined, "\n"))
	}
	for _, c := range corrections {
		gotLines += len(strings.Split(c.Msg, "\n"))
	}
	if gotLines != wantLines {
		t.Errorf("corrections describe %d lines of changes, diff has %d", gotLines, wantLines)
	}
}

// diff2 orders changes by their dependencies, not by kind: repointing a
// CNAME at a new host must create the host, then change the CNAME, then
// delete the old host. Grouping all deletes first would reverse that.
func TestGetZoneRecordsCorrectionsKeepsDependencyOrder(t *testing.T) {
	l, fake := newFakeProvider(t, nil)
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	existing := models.Records{
		newRC(t, dc, "x", "CNAME", "old.example.com."),
		newRC(t, dc, "old", "A", "1.1.1.1"),
	}
	dc.Records = models.Records{
		newRC(t, dc, "x", "CNAME", "new.example.com."),
		newRC(t, dc, "new", "A", "2.2.2.2"),
	}

	corrections, _, err := l.GetZoneRecordsCorrections(dc, existing)
	if err != nil {
		t.Fatal(err)
	}
	runAll(t, corrections)

	want := []recordedCall{
		{"POST", "/zones/5/records/create_many", []string{"new.example.com. A 2.2.2.2 300"}},
		{"PATCH", "/zones/5/records", []string{"x.example.com. CNAME new.example.com. 300"}},
		{"POST", "/zones/5/records/delete_many", []string{"old.example.com. A 1.1.1.1 300"}},
	}
	if got := fake.recorded(); !sameCalls(got, want) {
		t.Errorf("requests mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

// Large changesets are split into requests of at most maxBatchRRs records.
func TestGetZoneRecordsCorrectionsChunksLargeCreates(t *testing.T) {
	l, fake := newFakeProvider(t, nil)
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	const total = 250
	for i := range total {
		dc.Records = append(dc.Records, newRC(t, dc, fmt.Sprintf("h%d", i), "A", "192.0.2.1"))
	}

	corrections, _, err := l.GetZoneRecordsCorrections(dc, nil)
	if err != nil {
		t.Fatal(err)
	}
	runAll(t, corrections)

	calls := fake.recorded()
	if wantCalls := (total + maxBatchRRs - 1) / maxBatchRRs; len(calls) != wantCalls {
		t.Fatalf("got %d requests, want %d", len(calls), wantCalls)
	}
	sent := 0
	for _, c := range calls {
		if len(c.RRs) > maxBatchRRs {
			t.Errorf("request carries %d RRs, max is %d", len(c.RRs), maxBatchRRs)
		}
		sent += len(c.RRs)
	}
	if sent != total {
		t.Errorf("sent %d RRs, want %d", sent, total)
	}
}

// chunkChanges must keep a recordset's records together (UpdateManyRecords
// replaces the whole recordset it is given) and let an oversized recordset
// through on its own.
func TestChunkChanges(t *testing.T) {
	mk := func(sizes ...int) []pendingChange {
		var out []pendingChange
		for i, n := range sizes {
			out = append(out, pendingChange{rrs: make([]*api.RR, n), msg: fmt.Sprint(i)})
		}
		return out
	}
	shape := func(chunks [][]pendingChange) [][]string {
		var out [][]string
		for _, ch := range chunks {
			var msgs []string
			for _, c := range ch {
				msgs = append(msgs, c.msg)
			}
			out = append(out, msgs)
		}
		return out
	}

	tests := []struct {
		name  string
		sizes []int
		want  [][]string
	}{
		{"empty", nil, nil},
		{"exact fit", []int{2, 2}, [][]string{{"0", "1"}}},
		{"overflow starts new chunk", []int{2, 2, 1, 3}, [][]string{{"0", "1"}, {"2", "3"}}},
		{"oversized alone", []int{1, 5, 1}, [][]string{{"0"}, {"1"}, {"2"}}},
		{"oversized first", []int{5, 1}, [][]string{{"0"}, {"1"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shape(chunkChanges(mk(tc.sizes...), 4))
			if !slices.EqualFunc(got, tc.want, slices.Equal) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// A 500 on a create must not be retried (it may have been partly applied),
// while a 500 on a delete is retried.
func TestBulkRetryOn500(t *testing.T) {
	t.Run("create is not retried", func(t *testing.T) {
		l, fake := newFakeProvider(t, func(int) int { return http.StatusInternalServerError })
		dc, err := models.NewDomainConfig("example.com")
		if err != nil {
			t.Fatal(err)
		}
		dc.Records = models.Records{newRC(t, dc, "www", "A", "192.0.2.1")}
		corrections, _, err := l.GetZoneRecordsCorrections(dc, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(corrections) != 1 {
			t.Fatalf("got %d corrections, want 1", len(corrections))
		}
		if err := corrections[0].F(); err == nil {
			t.Error("500 on create was swallowed")
		}
		if n := len(fake.recorded()); n != 1 {
			t.Errorf("create sent %d times, want 1", n)
		}
	})

	t.Run("delete is retried", func(t *testing.T) {
		t.Parallel() // retry500 sleeps 2s before the first retry
		l, fake := newFakeProvider(t, func(n int) int {
			if n == 1 {
				return http.StatusInternalServerError
			}
			return http.StatusOK
		})
		dc, err := models.NewDomainConfig("example.com")
		if err != nil {
			t.Fatal(err)
		}
		existing := models.Records{newRC(t, dc, "www", "A", "192.0.2.1")}
		corrections, _, err := l.GetZoneRecordsCorrections(dc, existing)
		if err != nil {
			t.Fatal(err)
		}
		if len(corrections) != 1 {
			t.Fatalf("got %d corrections, want 1", len(corrections))
		}
		if err := corrections[0].F(); err != nil {
			t.Errorf("delete not retried after 500: %v", err)
		}
		if n := len(fake.recorded()); n != 2 {
			t.Errorf("delete sent %d times, want 2", n)
		}
	})
}

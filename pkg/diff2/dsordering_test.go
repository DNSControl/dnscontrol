package diff2

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// newRC is a small helper for building records in these tests.
func newRC(dc *models.DomainConfig, label, typ string, args ...any) *models.RecordConfig {
	rc, err := dc.NewRecordConfig(label, 300, typ, args...)
	if err != nil {
		panic(err)
	}
	return rc
}

// deleteThenCreateTypes returns the record types of the DELETE instructions
// followed by the record types of the CREATE instructions, in emitted order.
func changeTypesByVerb(changes ChangeList, verb Verb) []string {
	var out []string
	for _, c := range changes {
		if c.Type != verb {
			continue
		}
		switch verb {
		case DELETE:
			out = append(out, c.Old[0].Type)
		default:
			out = append(out, c.New[0].Type)
		}
	}
	return out
}

// TestDSNSOrdering verifies the DS<->NS dependency ordering. A DS delegation
// signs the NS at the same label, so:
//   - on create, the NS must be created before the DS, and
//   - on delete, the DS must be deleted before the NS.
//
// Multiple DS records at one label must NOT depend on each other (which would
// otherwise create a false circular dependency and leave them unordered).
func TestDSNSOrdering(t *testing.T) {
	dsDigest := "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	dsDigest2 := "ee02c885b5b4ed64899f2d43eb2b8e6619bdb50c"

	t.Run("delete DS before NS", func(t *testing.T) {
		dc := models.MustNewDomainConfig("example.com")
		existing := models.Records{
			newRC(dc, "child", "NS", "ns1.example.net."),
			newRC(dc, "child", "DS", uint16(1), uint8(13), uint8(1), dsDigest),
			newRC(dc, "child", "DS", uint16(2), uint8(13), uint8(1), dsDigest2),
		}
		dc.Records = models.Records{}

		changes, _, err := ByRecord(existing, dc, nil)
		if err != nil {
			t.Fatal(err)
		}
		order := changeTypesByVerb(changes, DELETE)
		assertBefore(t, order, "DS", "NS")
	})

	t.Run("create NS before DS", func(t *testing.T) {
		dc := models.MustNewDomainConfig("example.com")
		dc.Records = models.Records{
			newRC(dc, "child", "NS", "ns1.example.net."),
			newRC(dc, "child", "DS", uint16(1), uint8(13), uint8(1), dsDigest),
			newRC(dc, "child", "DS", uint16(2), uint8(13), uint8(1), dsDigest2),
		}

		changes, _, err := ByRecord(nil, dc, nil)
		if err != nil {
			t.Fatal(err)
		}
		order := changeTypesByVerb(changes, CREATE)
		assertBefore(t, order, "NS", "DS")
	})
}

// assertBefore fails if any occurrence of "after" precedes an occurrence of
// "before" in order. It also requires that both types are present.
func assertBefore(t *testing.T, order []string, before, after string) {
	t.Helper()
	lastBefore, firstAfter := -1, -1
	for i, typ := range order {
		switch typ {
		case before:
			lastBefore = i
		case after:
			if firstAfter == -1 {
				firstAfter = i
			}
		}
	}
	if lastBefore == -1 {
		t.Fatalf("no %s record found in order %v", before, order)
	}
	if firstAfter == -1 {
		t.Fatalf("no %s record found in order %v", after, order)
	}
	if lastBefore > firstAfter {
		t.Errorf("expected all %s before all %s, got order %v", before, after, order)
	}
}

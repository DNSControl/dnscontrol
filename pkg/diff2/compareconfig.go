package diff2

import (
	"fmt"
	"sort"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/prettyzone"
)

/*

Diffing the existing and desired DNS records is difficult. There are
many edge cases to consider.  The old pkg/diff system was successful
at handling all these edge cases but it made the code very complex.

pkg/diff2 was inspired by the intuition that the edge cases would
disappear if we simply stored the data in a way that was easier to
process. The edge cases would disappear and the code would become more
simple.  Simple is better.

The struct CompareConfig is a data structure that stores all the
existing/desired RecordConfig items in a way that makes the
differencing engine easier to implement. We extract the zone data
into this structure.

However, complexity never disappears it just moves elsewhere in the
system.  Converting our RecordConfigs to this datastructure is
complex. However by decoupling the conversion and the differencing, we
get two systems that can be tested independently. Thus, we have more
confidence in this system.

CompareConfig stores pointers to the original models.RecordConfig
grouped by label, and within each label grouped by rType (A, CNAME,
etc.).  In that final grouping the records are stored two ways. First,
as a list.  Second, as a list of (string,RecordConfig) tuples, where
the string is an opaque blob that can be used to compare for equality.
These lists are stored in an order that makes generating lists of
changes naturally be in the correct order for updates.

The structure also stores or pre-computes data that is needed by the
differencing engine, such as maps of which labels and RecordKeys
exist.
*/

// ComparableFunc is a signature for functions used to generate a comparable
// blob for custom records and records with metadata.
type ComparableFunc func(*models.RecordConfig) string

// CompareConfig stores a zone's records in a structure that makes comparing two zones convenient.
type CompareConfig struct {
	// The primary data. Each record stored once, grouped by label then
	// by rType:
	existing, desired models.Records // The original Recs.
	ldata             []*labelConfig // The Recs, grouped by label.
	//
	// Pre-computed values stored for easy access.
	origin   string                    // Domain zone
	labelMap map[string]bool           // Which labels exist?
	keyMap   map[models.RecordKey]bool // Which RecordKey exists?
	//
	// A function that generates a string used to compare two
	// RecordConfigs for equality.  This is normally nil. If it is not
	// nil, the function is called and the resulting string is joined to
	// the existing compareable string. This enables (for example)
	// custom rtypes to have their own comparison text added to the
	// comparison string.
	compareableFunc ComparableFunc
	//
}

type labelConfig struct {
	label string         // The label
	tdata []*rTypeConfig // The records for that label, grouped by rType.
}

type rTypeConfig struct {
	rType string // The rType for all records in this group (A, CNAME, etc)
	// The records stored as lists:
	existingRecs []*models.RecordConfig
	desiredRecs  []*models.RecordConfig
	// The records stored as compareable/rec tuples:
	existingTargets []targetConfig
	desiredTargets  []targetConfig
}

type targetConfig struct {
	comparableFull  string // A string that can be used to compare two rec's for exact equality.
	comparableNoTTL string // A string that can be used to compare two rec's for equality, ignoring the TTL

	rec *models.RecordConfig // The RecordConfig itself.
}

// NewCompareConfig creates a CompareConfig from a set of records and other data.
func NewCompareConfig(origin string, existing, desired models.Records, compFn ComparableFunc) *CompareConfig {
	cc := &CompareConfig{
		existing: existing,
		desired:  desired,
		//
		origin:          origin,
		compareableFunc: compFn,
		//
		labelMap: map[string]bool{},
		keyMap:   map[models.RecordKey]bool{},
	}
	cc.addRecords(existing, true) // Must be called first so that CNAME manipulations happen in the correct order.
	cc.addRecords(desired, false)
	cc.verifyCNAMEAssertions()
	sort.Slice(cc.ldata, func(i, j int) bool {
		return prettyzone.LabelLess(cc.ldata[i].label, cc.ldata[j].label)
	})
	return cc
}

// verifyCNAMEAssertions verifies assertions about CNAME updates ordering.
func (cc *CompareConfig) verifyCNAMEAssertions() {
	// According to the RFCs if a label has a CNAME, it can not have any other
	// records at that label... even other CNAMEs.  Therefore, we need to be
	// careful with changes at a label that involve a CNAME.
	// Example 1:
	//   OLD: a.example.com CNAME b
	//   NEW: a.example.com A 1.2.3.4
	//   We must delete the CNAME record THEN create the A record.  If we
	//   blindly create the A first, most APIs will reply with an error
	//   because there is already a CNAME at that label.
	// Example 2:
	//   OLD: a.example.com A 1.2.3.4
	//   NEW: a.example.com CNAME b
	//   We must delete the A record THEN create the CNAME.  If we blindly
	//   create the CNAME first, most APIs will reply with an error because
	//   there is already an A record at that label.
	//
	// To assure that DNS providers don't have to think about this, we order
	// the tdata items so that we generate the instructions in the correct order.
	// In other words:
	//     If there is a CNAME in existing, it should be in front.
	//     If there is a CNAME in desired, it should be at the end.

	// That said, since we addRecords existing first, and desired last, the data
	// should already be in the right order.

	for _, ld := range cc.ldata {
		for j, td := range ld.tdata {
			if td.rType == "CNAME" {
				// This assertion doesn't hold for a site that permits a
				// recordset with both CNAMEs and other records, such as
				// Cloudflare.
				// Therefore, we skip the test if we aren't deleting
				// everything at the recordset or creating it from scratch.
				if len(td.existingTargets) != 0 && len(td.desiredTargets) != 0 {
					continue
				}

				if len(td.existingTargets) != 0 {
					if j != 0 {
						fmt.Println("WARNING: should not happen: (CNAME not in FIRST position)")
					}
				}

				if len(td.desiredTargets) != 0 {
					if j != highest(ld.tdata) {
						fmt.Println("WARNING: should not happen: (CNAME not in last position). There are multiple records on a label with a CNAME.")
					}
				}
			}
		}
	}
}

// Generate a string that can be used to compare this record to others
// for equality.
func mkCompareBlobs(rc *models.RecordConfig, f func(*models.RecordConfig) string) (string, string) {
	// Start with the comparable string
	comp := rc.ToComparableNoTTL()

	// If the custom function exists, add its output
	if f != nil {
		addOn := f(rc)
		if addOn != "" {
			comp += " " + f(rc)
		}
	}

	// We do this to save memory. This assures the first return value uses the same memory as the second.
	lenWithoutTTL := len(comp)
	compFull := comp + fmt.Sprintf(" ttl=%d", rc.TTL)

	return compFull[:lenWithoutTTL], compFull
}

func (cc *CompareConfig) addRecords(recs models.Records, storeInExisting bool) {
	// storeInExisting indicates if the records should be stored in the
	// cc.existing* fields (true) or the cc.desired* fields (false).

	// Sort, because sorted data is easier to work with.
	// NB(tlim): The actual sort order doesn't matter as long as all the records
	// of the same label+rtype are grouped. We use PrettySort because it works,
	// has been extensively tested, and assures that the ChangeList will
	// be in an order that is pretty to look at.
	z := prettyzone.PrettySort(recs, cc.origin, 0, nil)

	for _, rec := range z.Records {
		key := rec.Key()
		label := key.NameFQDN
		rtype := key.Type
		compNoTTL, compFull := mkCompareBlobs(rec, cc.compareableFunc)

		// Are we seeing this label for the first time?
		var labelIdx int
		if _, ok := cc.labelMap[label]; !ok {
			cc.labelMap[label] = true
			cc.ldata = append(cc.ldata, &labelConfig{label: label})
			labelIdx = highest(cc.ldata)
		} else {
			for k, v := range cc.ldata {
				if v.label == label {
					labelIdx = k
					break
				}
			}
		}

		// Are we seeing this label+rtype for the first time?
		if _, ok := cc.keyMap[key]; !ok {
			cc.keyMap[key] = true
			x := cc.ldata[labelIdx]
			x.tdata = append(x.tdata, &rTypeConfig{rType: rtype})
		}
		var rtIdx int
		// find rtype in tdata:
		for k, v := range cc.ldata[labelIdx].tdata {
			if v.rType == rtype {
				rtIdx = k
				break
			}
		}

		// Now it is safe to add/modify the records.

		if storeInExisting {
			cc.ldata[labelIdx].tdata[rtIdx].existingRecs = append(cc.ldata[labelIdx].tdata[rtIdx].existingRecs, rec)
			cc.ldata[labelIdx].tdata[rtIdx].existingTargets = append(cc.ldata[labelIdx].tdata[rtIdx].existingTargets,
				targetConfig{comparableNoTTL: compNoTTL, comparableFull: compFull, rec: rec})
		} else {
			cc.ldata[labelIdx].tdata[rtIdx].desiredRecs = append(cc.ldata[labelIdx].tdata[rtIdx].desiredRecs, rec)
			cc.ldata[labelIdx].tdata[rtIdx].desiredTargets = append(cc.ldata[labelIdx].tdata[rtIdx].desiredTargets,
				targetConfig{comparableNoTTL: compNoTTL, comparableFull: compFull, rec: rec})
		}
	}
}

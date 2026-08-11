package diff2

//go:generate go tool stringer -type=Verb

// This module provides functions that "diff" the existing records
// against the desired records.

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnsgraph"
)

// Verb indicates the Change's type (create, delete, etc.)
type Verb int

// CREATE and other verbs.
const (
	_      Verb = iota // Skip the first value of 0
	CREATE             // Create a record/recordset/label where none existed before.
	CHANGE             // Change existing record/recordset/label
	DELETE             // Delete existing record/recordset/label
	REPORT             // No change, but I have something to say!
)

// ChangeList is a list of Change.
type ChangeList []Change

// Change is an instruction to the provider. Generally if one properly executes
// all the changes, an "existing" zone will turn into the "desired" zone.
type Change struct {
	Type Verb // Add, Change, Delete

	Key        models.RecordKey // .Key.Type is "" unless using ByRecordSet
	Old        models.Records
	New        models.Records                // any changed or added records at Key.
	Msgs       []string                      // Human-friendly explanation of what changed
	MsgsJoined string                        // strings.Join(Msgs, "\n")
	MsgsByKey  map[models.RecordKey][]string // Messages for a given key

	// HintOnlyTTL is true only if (.Type == diff2.CHANGE) && (there is
	// exactly 1 record being updated) && (the only change is the TTL)
	HintOnlyTTL bool
}

// GetType returns the type of a change.
func (c Change) GetType() dnsgraph.NodeType {
	if c.Type == REPORT {
		return dnsgraph.Report
	}

	return dnsgraph.Change
}

// GetName returns the FQDN of the host being changed.
func (c Change) GetName() string {
	return c.Key.NameFQDN
}

// GetDependencies returns the depenencies of a change.
func (c Change) GetDependencies() []dnsgraph.Dependency {
	var dependencies []dnsgraph.Dependency

	if c.Type == CHANGE || c.Type == DELETE {
		dependencies = append(dependencies, dnsgraph.CreateDependencies(c.Old.GetAllDependencies(), dnsgraph.BackwardDependency)...)
	}
	if c.Type == CHANGE || c.Type == CREATE {
		dependencies = append(dependencies, dnsgraph.CreateDependencies(c.New.GetAllDependencies(), dnsgraph.ForwardDependency)...)
	}

	return dependencies
}

/*
General instructions:

Every provider updates their zones in one of three granularities: record, recordset, label, or zone.
Identify which your provider is, and use the appropriate "By*" call. (ByZone is different, see its documentation.)

  changes, err := diff2.ByRecord(existing, dc, nil)
  //changes, err := diff2.ByRecordSet(existing, dc, nil)
  //changes, err := diff2.ByLabel(existing, dc, nil)
  if err != nil {
    return nil, err
  }

  var corrections []*models.Correction

  for _, change := range changes {
    switch change.Type {
    case diff2.REPORT:
      corr = change.CreateMessage()
    case diff2.CREATE:
      corr = change.CreateCorrection(func() error { return c.createRecord(FILL_IN) })
    case diff2.CHANGE:
      corr = change.CreateCorrection(func() error { return c.modifyRecord(FILL_IN) })
    case diff2.DELETE:
      corr = change.CreateCorrection(func() error { return c.deleteRecord(FILL_IN) })
	  default:
		  panic("unhandled change.TYPE %s", change.Type)
    }

    corrections = append(corrections, corr)
  }

  return corrections, nil
}

*/

// CreateCorrection creates a new Correction based on the given
// function and prefills it with the Msg of the current Change.
func (c *Change) CreateCorrection(correctionFunction func() error) *models.Correction {
	return &models.Correction{
		F:   correctionFunction,
		Msg: c.MsgsJoined,
	}
}

// CreateMessage creates a new correction with only the message.
// Used for diff2.Report corrections.
func (c *Change) CreateMessage() *models.Correction {
	return &models.Correction{
		Msg: c.MsgsJoined,
	}
}

// CreateCorrectionWithMessage creates a new Correction based on the
// given function and prefixes given function with the Msg of the
// current change.
func (c *Change) CreateCorrectionWithMessage(msg string, correctionFunction func() error) *models.Correction {
	return &models.Correction{
		F:   correctionFunction,
		Msg: fmt.Sprintf("%s: %s", msg, c.MsgsJoined),
	}
}

// ByRecordSet takes two lists of records (existing and desired) and
// returns instructions for turning existing into desired.
//
// Use this with DNS providers whose API updates one recordset at a
// time. A recordset is all the records of a particular type at a
// label. For example, if www.example.com has 3 A records and a TXT
// record, if A records are added, changed, or removed, the API takes
// www.example.com, A, and a list of all the desired IP addresses.
//
// Examples include: AZURE_DNS, GCORE, NS1, ROUTE53.
func ByRecordSet(existing models.Records, dc *models.DomainConfig, compFunc ComparableFunc) (ChangeList, int, error) {
	return byHelper(analyzeByRecordSet, existing, dc, compFunc)
}

// ByLabel takes two lists of records (existing and desired) and
// returns instructions for turning existing into desired.
//
// Use this with DNS providers whose API updates one label at a
// time. That is, updates are done by sending a list of DNS records
// to be served at a particular label, or the label itself is deleted.
//
// Examples include: GANDI_V5.
func ByLabel(existing models.Records, dc *models.DomainConfig, compFunc ComparableFunc) (ChangeList, int, error) {
	return byHelper(analyzeByLabel, existing, dc, compFunc)
}

// ByRecord takes two lists of records (existing and desired) and
// returns instructions for turning existing into desired.
//
// Use this with DNS providers whose API updates one record at a time.
//
// Note: The .Old and .New field are lists ([]models.RecordConfig) but
// when using ByRecord() they will never have more than one entry.
// A create always has exactly 1 new: .New[0]
// A change always has exactly 1 old and 1 new: .Old[0] and .New[0]
// A delete always has exactly 1 old: .Old[0]
//
// Examples include: CLOUDFLAREAPI, HEDNS, INWX, MSDNS, OVH, PORKBUN, VULTR.
func ByRecord(existing models.Records, dc *models.DomainConfig, compFunc ComparableFunc) (ChangeList, int, error) {
	return byHelper(analyzeByRecord, existing, dc, compFunc)
}

// ByZone takes two lists of records (existing and desired) and
// returns text to output to users describing the change, a bool
// indicating if there were any changes, and a possible err value.
//
// Use this with DNS providers whose API updates the entire zone at a
// time. That is, to make any change (even just 1 record) the entire DNS
// zone is uploaded.
//
// The user should see a list of changes as if individual records were updated.
//
// Example usage:
//
//	result, err := diff2.ByZone(foundRecords, dc, nil)
//	if err != nil {
//	  return nil, err
//	}
//	if result.HasChanges {
//		// Generate a "correction" that uploads the entire zone.
//		// (result.DesiredPlus are the new records for the zone).
//	}
//
// Example providers include: BIND, AUTODNS.
func ByZone(existing models.Records, dc *models.DomainConfig, compFunc ComparableFunc) (ByResults, error) {
	// Only return the messages and a list of records needed to build the new zone.
	result, err := byHelperStruct(analyzeByRecord, existing, dc, compFunc)
	result.Msgs = justMsgs(result.Instructions)
	return result, err
}

// ByResults is the results of ByZone() and perhaps someday all the By*() functions.
// It is partially populated by byHelperStruct() and partially by the By*()
// functions that use it.
type ByResults struct {
	// Fields filled in by byHelperStruct():
	Instructions      ChangeList     // Instructions to turn existing into desired.
	ActualChangeCount int            // Number of actual changes, not including REPORTs.
	HasChanges        bool           // True if there are any changes.
	DesiredPlus       models.Records // Desired + foreign + ignored
	// Fields filled in by ByZone():
	Msgs []string // Just the messages from the instructions.
}

// byHelper is like byHelperStruct but has a signature that is compatible with legacy code.
// Deprecated: Use byHelperStruct instead.
func byHelper(fn func(cc *CompareConfig) (ChangeList, int), existing models.Records, dc *models.DomainConfig, compFunc ComparableFunc) (ChangeList, int, error) {
	result, err := byHelperStruct(fn, existing, dc, compFunc)
	return result.Instructions, result.ActualChangeCount, err
}

// byHelperStruct does 90% of the work for the By*() calls.
func byHelperStruct(fn func(cc *CompareConfig) (ChangeList, int), existing models.Records, dc *models.DomainConfig, compFunc ComparableFunc) (ByResults, error) {
	// Process NO_PURGE/ENSURE_ABSENT and IGNORE*().
	desiredPlus, msgs, err := handsoff(
		dc.Name,
		existing, dc.Records, dc.EnsureAbsent,
		dc.Unmanaged,
		dc.UnmanagedUnsafe,
		dc.KeepUnknown,
		dc.IgnoreExternalDNS,
		dc.ExternalDNSPrefix,
	)
	if err != nil {
		return ByResults{}, err
	}

	// Regroup existing/desiredd for easy comparison:
	cc := NewCompareConfig(dc.Name, existing, desiredPlus, compFunc)

	// Analyze and generate the instructions:
	instructions, actualChangeCount := fn(cc)

	// If we have msgs, create a change to output them:
	if len(msgs) != 0 {
		chg := Change{
			Type:       REPORT,
			Msgs:       msgs,
			MsgsJoined: strings.Join(msgs, "\n"),
		}
		_ = chg
		instructions = append([]Change{chg}, instructions...)
	}

	return ByResults{
		Instructions:      instructions,
		ActualChangeCount: actualChangeCount,
		HasChanges:        actualChangeCount > 0,
		DesiredPlus:       desiredPlus,
	}, nil
}

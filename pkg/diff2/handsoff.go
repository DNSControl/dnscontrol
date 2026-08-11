package diff2

// This file implements the features that tell DNSControl "hands off"
// foreign-controlled (or shared-control) DNS records.  i.e. the
// NO_PURGE, ENSURE_ABSENT and IGNORE*() features.

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/gobwas/glob"
)

/*

# How do NO_PURGE, IGNORE*() and ENSURE_ABSENT work?

## Terminology:

* "existing" refers to the records downloaded from the provider via the API.
* "desired" refers to the records generated from dnsconfig.js.
* "absences" refers to a list of records tagged with ENSURE_ABSENT.

## What are the features?

There are 2 ways to tell DNSControl not to touch existing records in a domain,
and 1 way to make exceptions.

* NO_PURGE: Tells DNSControl not to delete records in a domain.
	* New records will be created
	* Existing records (matched on label:rtype) will be modified.
	* FYI: This means you can't have a label with two A records, one controlled
	    by DNSControl and one controlled by an external system.
* IGNORE(labelglob, typelist, targetglob):
    * "If an existing record matches this pattern, don't touch it!""
    * IGNORE_NAME(foo, bar) is the same as IGNORE(foo, bar, "*")
    * IGNORE_TARGET(foo) is the same as IGNORE("*", "*", foo)
    * FYI: You CAN have a label with two A records, one controlled by
	    DNSControl and one controlled by an external system.  DNSControl would
		need to have an IGNORE() statement with a targetglob that matches
	    the external system's target values.
* ENSURE_ABSENT: Override NO_PURGE for specific records. i.e. delete them even
    though NO_PURGE is enabled.
    * If any of these records are in desired (matched on
      label:rtype:target), remove them.  This takes priority over
      NO_PURGE/IGNORE*().

## Implementation premise

The fundamental premise is "if you don't want it deleted, copy it to the
'desired' list." So, for example, if you want to IGNORE_NAME("www"), then you
find any records with the label "www" in "existing" and copy them to "desired".
As a result, the diff2 algorithm won't delete them because they are desired!
(Of course "desired" can't have duplicate records. Check before you add.)

This is different than in the old implementation (pkg/diff) which would generate the
diff but then do a bunch of checking to see if the record was one that
shouldn't be deleted.  Or, in the case of NO_PURGE, would simply not do the
deletions.  This was complex because there were many edge cases to deal with.
It was often also wrong. For example, if a provider updates all records in a
RecordSet at once, you shouldn't NOT update the record.

## Implementation

Here is how we intend to implement these features:

  IGNORE() is implemented as:
  * Take the list of existing records. If any match one of the IGNORE glob
      patterns, add it to the "ignored list".
  * If any item on the "ignored list" is also in "desired" (match on
      label:rtype), output a warning (defeault) or declare an error (if
      DISABLE_IGNORE_SAFETY_CHECK is true).
  * When we're done, add the "ignore list" records to desired.

  NO_PURGE + ENSURE_ABSENT is implemented as:
  * Take the list of existing records. If any do not appear in desired, add them
      to desired UNLESS they appear in absences. (Yes, that's complex!)
  * "appear in desired" is done by matching on label:type.
  * "appear in absences" is done by matching on label:type:target.

The actual implementation combines this all into one loop:
    foreach rec in existing:
        if rec matches_any_unmanaged_pattern:
            if rec in desired:
                if "DISABLE_IGNORE_SAFETY_CHECK" is false:
                    Display a warning.
                else
                    Return an error.
            Add rec to "ignored list"
        else:
            if NO_PURGE:
                if rec NOT in desired: (matched on label:type)
                    if rec NOT in absences: (matched on label:type:target)
                        Add rec to "foreign list"
    Append "ignored list" to "desired".
    Append "foreign list" to "desired".
*/

// handsoff processes the IGNORE*()//NO_PURGE/ENSURE_ABSENT features.
func handsoff(
	domain string,
	existing, desired, absences models.Records,
	unmanagedConfigs []*models.UnmanagedConfig,
	unmanagedSafely bool,
	noPurge bool,
	ignoreExternalDNS bool,
	externalDNSPrefix string,
) (models.Records, []string, error) {
	var msgs []string

	// Prep the globs:
	err := compileUnmanagedConfigs(unmanagedConfigs)
	if err != nil {
		return nil, nil, err
	}

	punct := ":"
	if printer.MaxReport == 0 {
		punct = "."
	}

	// Process IGNORE_EXTERNAL_DNS feature:
	var externalDNSIgnored models.Records
	if ignoreExternalDNS {
		externalDNSIgnored = GetExternalDNSIgnoredRecords(existing, domain, externalDNSPrefix)
		if len(externalDNSIgnored) != 0 {
			msgs = append(msgs, fmt.Sprintf("%d records not being deleted because of IGNORE_EXTERNAL_DNS%s", len(externalDNSIgnored), punct))
			msgs = append(msgs, reportSkips(externalDNSIgnored, !printer.SkinnyReport)...)
		}
	}

	// Process IGNORE*() and NO_PURGE features:
	ignorable, foreign, err := processIgnoreAndNoPurge(domain, existing, desired, absences, unmanagedConfigs, noPurge)
	if err != nil {
		return nil, nil, err
	}
	if len(foreign) != 0 {
		msgs = append(msgs, fmt.Sprintf("%d records not being deleted because of NO_PURGE%s", len(foreign), punct))
		msgs = append(msgs, reportSkips(foreign, !printer.SkinnyReport)...)
	}
	if len(ignorable) != 0 {
		msgs = append(msgs, fmt.Sprintf("%d records not being deleted because of IGNORE*()%s", len(ignorable), punct))
		msgs = append(msgs, reportSkips(ignorable, !printer.SkinnyReport)...)
	}

	// Check for invalid use of IGNORE_*.
	conflicts := findConflicts(unmanagedConfigs, desired)
	if len(conflicts) != 0 {
		msgs = append(msgs, fmt.Sprintf("%d records that are both IGNORE*()'d and not ignored:", len(conflicts)))
		for _, r := range conflicts {
			msgs = append(msgs, fmt.Sprintf("    %s %s %s", r.GetLabelFQDN(), r.Type, r.GetTargetCombined()))
		}
		if !unmanagedSafely {
			return nil, nil, errors.New(strings.Join(msgs, "\n") +
				"\nERROR: Unsafe to continue. Add DISABLE_IGNORE_SAFETY_CHECK to D() to override")
		}
	}

	// Check for conflicts between desired records and external-dns managed records.
	// This warns users when they define a record that external-dns is also managing.
	if ignoreExternalDNS && len(externalDNSIgnored) > 0 {
		externalDNSConflicts := findExternalDNSConflicts(desired, externalDNSIgnored)
		if len(externalDNSConflicts) != 0 {
			msgs = append(msgs, fmt.Sprintf("WARNING: %d records are defined in your config but also managed by external-dns:", len(externalDNSConflicts)))
			for _, r := range externalDNSConflicts {
				msgs = append(msgs, fmt.Sprintf("    %s %s %s", r.GetLabelFQDN(), r.Type, r.GetTargetCombined()))
			}
			msgs = append(msgs, "Consider removing these from your config or from external-dns to avoid conflicts.")
		}
		// Filter out conflicts from externalDNSIgnored to avoid duplicates in desired
		externalDNSIgnored = filterOutConflicts(externalDNSIgnored, externalDNSConflicts)
	}

	// Add the ignored/foreign items to the desired list so they are not deleted:
	desired = append(desired, ignorable...)
	desired = append(desired, foreign...)
	desired = append(desired, externalDNSIgnored...)
	return desired, msgs, nil
}

// reportSkips reports records being skipped, if !full only the first
// printer.MaxReport are output.
func reportSkips(recs models.Records, full bool) []string {
	var msgs []string

	shorten := (!full) && (len(recs) > printer.MaxReport)

	last := len(recs)
	if shorten {
		last = printer.MaxReport
	}

	for _, r := range recs[:last] {
		msgs = append(msgs, fmt.Sprintf(`    %s("%s.", %s),`, r.Type, r.GetLabelFQDN(), r.GetTargetJS()))
	}
	if shorten && printer.MaxReport != 0 {
		msgs = append(msgs, fmt.Sprintf("    ...and %d more... (use --full to show all)", len(recs)-printer.MaxReport))
	}

	return msgs
}

// processIgnoreAndNoPurge processes the IGNORE_*() and NO_PURGE/ENSURE_ABSENT() features.
func processIgnoreAndNoPurge(domain string, existing, desired, absences models.Records, unmanagedConfigs []*models.UnmanagedConfig, noPurge bool) (models.Records, models.Records, error) {
	var ignorable, foreign models.Records
	desiredDB := models.NewRecordDBFromRecords(desired, domain)
	absentDB := models.NewRecordDBFromRecords(absences, domain)
	if err := compileUnmanagedConfigs(unmanagedConfigs); err != nil {
		return nil, nil, err
	}
	for _, rec := range existing {
		isMatch := matchAny(unmanagedConfigs, rec)
		// fmt.Printf("DEBUG: matchAny returned: %v\n", isMatch)
		if isMatch {
			ignorable = append(ignorable, rec)
		} else {
			if noPurge {
				// Is this a candidate for purging?
				if !desiredDB.ContainsLT(rec) {
					// Yes, but not if it is an exception!
					if !absentDB.ContainsLT(rec) {
						foreign = append(foreign, rec)
					}
				}
			}
		}
	}
	return ignorable, foreign, nil
}

// findConflicts takes a list of recs and a list of (compiled) UnmanagedConfigs
// and reports if any of the recs match any of the configs.
func findConflicts(uconfigs []*models.UnmanagedConfig, recs models.Records) models.Records {
	var conflicts models.Records
	for _, rec := range recs {
		if matchAny(uconfigs, rec) {
			conflicts = append(conflicts, rec)
		}
	}
	return conflicts
}

// compileUnmanagedConfigs prepares a slice of UnmanagedConfigs so they can be used.
func compileUnmanagedConfigs(configs []*models.UnmanagedConfig) error {
	var err error

	for i := range configs {
		c := configs[i]

		if c.LabelPattern == "" || c.LabelPattern == "*" {
			c.LabelGlob = nil // nil indicates "always match"
		} else {
			c.LabelGlob, err = glob.Compile(c.LabelPattern)
			if err != nil {
				return err
			}
		}

		c.RTypeMap = make(map[string]struct{})
		if c.RTypePattern != "*" && c.RTypePattern != "" {
			for part := range strings.SplitSeq(c.RTypePattern, ",") {
				part = strings.TrimSpace(part)
				c.RTypeMap[part] = struct{}{}
			}
		}

		if c.TargetPattern == "" || c.TargetPattern == "*" {
			c.TargetGlob = nil // nil indicates "always match"
		} else {
			c.TargetGlob, err = glob.Compile(c.TargetPattern)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// matchAny returns true if rec matches any of the uconfigs.
func matchAny(uconfigs []*models.UnmanagedConfig, rec *models.RecordConfig) bool {
	// fmt.Printf("DEBUG: matchAny(%s, %q, %q, %q)\n", models.DebugUnmanagedConfig(uconfigs), rec.NameFQDN, rec.Type, rec.GetTargetField())
	for _, uc := range uconfigs {
		if matchLabel(uc.LabelGlob, rec.GetLabel()) &&
			matchType(uc.RTypeMap, rec.Type) &&
			matchTarget(uc.TargetGlob, rec.GetTargetField()) {
			return true
		}
	}
	return false
}

func matchLabel(labelGlob glob.Glob, labelName string) bool {
	if labelGlob == nil {
		return true
	}
	return labelGlob.Match(labelName)
}

func matchType(typeMap map[string]struct{}, typeName string) bool {
	if len(typeMap) == 0 {
		return true
	}
	_, ok := typeMap[typeName]
	return ok
}

func matchTarget(targetGlob glob.Glob, targetName string) bool {
	if targetGlob == nil {
		return true
	}
	return targetGlob.Match(targetName)
}

// findExternalDNSConflicts returns records that appear in both desired and externalDNSIgnored.
// This helps identify when a user has defined a record in their config that is also
// being managed by external-dns.
func findExternalDNSConflicts(desired, externalDNSIgnored models.Records) models.Records {
	// Build a map of desired records keyed by label:type
	desiredMap := make(map[string]bool)
	for _, rec := range desired {
		key := rec.GetLabel() + ":" + rec.Type
		desiredMap[key] = true
	}

	// Find any external-dns ignored records that are also in desired
	var conflicts models.Records
	for _, rec := range externalDNSIgnored {
		key := rec.GetLabel() + ":" + rec.Type
		if desiredMap[key] {
			conflicts = append(conflicts, rec)
		}
	}
	return conflicts
}

// filterOutConflicts removes records from externalDNSIgnored that are in conflicts.
// This prevents duplicates when appending externalDNSIgnored to desired.
func filterOutConflicts(externalDNSIgnored, conflicts models.Records) models.Records {
	if len(conflicts) == 0 {
		return externalDNSIgnored
	}

	// Build a set of conflict keys
	conflictSet := make(map[string]bool)
	for _, rec := range conflicts {
		key := rec.GetLabel() + ":" + rec.Type
		conflictSet[key] = true
	}

	// Filter out conflicts
	var filtered models.Records
	for _, rec := range externalDNSIgnored {
		key := rec.GetLabel() + ":" + rec.Type
		if !conflictSet[key] {
			filtered = append(filtered, rec)
		}
	}
	return filtered
}

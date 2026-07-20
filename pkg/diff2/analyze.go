package diff2

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/fatih/color"
)

func analyzeByRecordSet(cc *CompareConfig) (ChangeList, int) {
	var instructions ChangeList
	var actualChangeCount int
	// For each label...
	for _, lc := range cc.ldata {
		// for each type at that label...
		for _, rt := range lc.tdata {
			// ...if there are changes generate an instruction.
			ets := rt.existingTargets
			dts := rt.desiredTargets
			cs := diffTargets(ets, dts)
			actualChangeCount += len(cs)
			msgs := justMsgs(cs)
			if len(msgs) == 0 { // No differences?
				// The records at this rset are the same. No work to be done.
				continue
			}
			if len(ets) == 0 { // Create a new label.
				instructions = append(instructions, mkAdd(lc.label, rt.rType, msgs, rt.desiredRecs))
			} else if len(dts) == 0 { // Delete that label and all its records.
				instructions = append(instructions, mkDelete(lc.label, rt.rType, msgs, rt.existingRecs))
			} else { // Change the records at that label
				instructions = append(instructions, mkChange(lc.label, rt.rType, msgs, rt.existingRecs, rt.desiredRecs))
			}
		}
	}

	instructions = orderByDependencies(instructions)

	return instructions, actualChangeCount
}

func analyzeByLabel(cc *CompareConfig) (ChangeList, int) {
	var instructions ChangeList
	var actualChangeCount int
	// Accumulate any changes and collect the info needed to generate instructions.
	for _, lc := range cc.ldata {
		// for each type at that label...
		label := lc.label
		var accMsgs []string
		var accExisting models.Records
		var accDesired models.Records
		msgsByKey := map[models.RecordKey][]string{}
		for _, rt := range lc.tdata {
			// for each type at that label...
			ets := rt.existingTargets
			dts := rt.desiredTargets
			cs := diffTargets(ets, dts)
			actualChangeCount += len(cs)
			msgs := justMsgs(cs)
			k := models.RecordKey{NameFQDN: label, Type: rt.rType}
			msgsByKey[k] = msgs
			accMsgs = append(accMsgs, msgs...)                    // Accumulate the messages
			accExisting = append(accExisting, rt.existingRecs...) // Accumulate records existing at this label.
			accDesired = append(accDesired, rt.desiredRecs...)    // Accumulate records desired at this label.
		}

		// We now know what changed (accMsgs),
		// what records USED TO EXIST at that label (accExisting),
		// and what records SHOULD EXIST at that label (accDesired).
		// Based on that info, we can generate the instructions.

		if len(accMsgs) == 0 { // Nothing changed.
		} else if len(accDesired) == 0 { // No new records at the label? This must be a delete.
			instructions = append(instructions, mkDelete(label, "", accMsgs, accExisting))
		} else if len(accExisting) == 0 { // No old records at the label? This must be a change.
			c := mkAdd(label, "", accMsgs, accDesired)
			c.MsgsByKey = msgsByKey
			instructions = append(instructions, c)
		} else { // If we get here, it must be a change.
			instructions = append(instructions, mkChange(label, "", accMsgs, accExisting, accDesired))
		}
	}

	instructions = orderByDependencies(instructions)

	return instructions, actualChangeCount
}

func analyzeByRecord(cc *CompareConfig) (ChangeList, int) {
	var instructions ChangeList
	var actualChangeCount int
	// For each label, for each type at that label, see if there are any changes.
	for _, lc := range cc.ldata {
		for _, rt := range lc.tdata {
			ets := rt.existingTargets
			dts := rt.desiredTargets
			cs := diffTargets(ets, dts)
			actualChangeCount += len(cs)
			instructions = append(instructions, cs...)
		}
	}

	instructions = orderByDependencies(instructions)

	return instructions, actualChangeCount
}

// FYI: there is no analyzeByZone.  diff2.ByZone() calls analyzeByRecords().

func mkAdd(l string, t string, msgs []string, newRecs models.Records) Change {
	c := Change{Type: CREATE, Msgs: msgs, MsgsJoined: strings.Join(msgs, "\n")}
	c.Key.NameFQDN = l
	c.Key.Type = t
	c.New = newRecs
	return c
}

func mkChange(l string, t string, msgs []string, oldRecs, newRecs models.Records) Change {
	c := Change{Type: CHANGE, Msgs: msgs, MsgsJoined: strings.Join(msgs, "\n")}
	c.Key.NameFQDN = l
	c.Key.Type = t
	c.Old = oldRecs
	c.New = newRecs
	return c
}

func mkDelete(l string, t string, msgs []string, oldRecs models.Records) Change {
	c := Change{Type: DELETE, Msgs: msgs, MsgsJoined: strings.Join(msgs, "\n")}
	c.Key.NameFQDN = l
	c.Key.Type = t
	c.Old = oldRecs
	return c
}

func removeCommon(existing, desired []targetConfig) ([]targetConfig, []targetConfig) {
	// Sort by comparableFull.
	sort.Slice(existing, func(i, j int) bool { return existing[i].comparableFull < existing[j].comparableFull })
	sort.Slice(desired, func(i, j int) bool { return desired[i].comparableFull < desired[j].comparableFull })

	// Build maps required by filterBy
	eKeys := map[string]*targetConfig{}
	for _, v := range existing {
		eKeys[v.comparableFull] = &v
	}
	dKeys := map[string]*targetConfig{}
	for _, v := range desired {
		dKeys[v.comparableFull] = &v
	}

	return filterBy(existing, dKeys), filterBy(desired, eKeys)
}

// findTTLChanges finds the records that ONLY change their TTL. For those, generate a Change.
// Remove such items from the list.
func findTTLChanges(existing, desired []targetConfig) ([]targetConfig, []targetConfig, ChangeList) {
	if (len(existing) == 0) || (len(desired) == 0) {
		return existing, desired, nil
	}

	// Sort by comparableNoTTL
	sort.Slice(existing, func(i, j int) bool { return existing[i].comparableNoTTL < existing[j].comparableNoTTL })
	sort.Slice(desired, func(i, j int) bool { return desired[i].comparableNoTTL < desired[j].comparableNoTTL })

	var instructions ChangeList
	var existDiff, desiredDiff []targetConfig
	ei := 0
	di := 0
	for (ei < len(existing)) && (di < len(desired)) {
		er := existing[ei].rec
		dr := desired[di].rec
		ecomp := existing[ei].comparableNoTTL
		dcomp := desired[di].comparableNoTTL

		if ecomp == dcomp && er.TTL == dr.TTL {
			panic(fmt.Sprintf(
				"Should not happen. There should be some difference! ecomp=%q dcomp=%q er.TTL=%v dr.TTL=%v\n",
				ecomp, dcomp, er.TTL, dr.TTL))
		}

		if ecomp == dcomp && er.TTL != dr.TTL {
			m := color.YellowString("± MODIFY-TTL %s %s %s", dr.NameFQDN, dr.Type, humanDiff(existing[ei], desired[di]))
			v := mkChange(dr.NameFQDN, dr.Type, []string{m},
				models.Records{er},
				models.Records{dr},
			)
			v.HintOnlyTTL = true
			instructions = append(instructions, v)
			ei++
			di++
		} else if ecomp < dcomp {
			existDiff = append(existDiff, existing[ei])
			ei++
		} else if ecomp > dcomp {
			desiredDiff = append(desiredDiff, desired[di])
			di++
		} else {
			panic("should not happen. ecomp cant be both == and != dcomp")
		}
	}

	// Any remainder goes to the *Diff result:
	if ei < len(existing) {
		existDiff = append(existDiff, existing[ei:]...)
	}
	if di < len(desired) {
		desiredDiff = append(desiredDiff, desired[di:]...)
	}

	return existDiff, desiredDiff, instructions
}

// Return s but remove any items that can be found in m.
func filterBy(s []targetConfig, m map[string]*targetConfig) []targetConfig {
	i := 0 // output index
	for _, x := range s {
		if _, ok := m[x.comparableFull]; !ok {
			// copy and increment index
			s[i] = x
			i++
		}
	}
	s = s[:i]
	return s
}

// humanDiff returns a human-friendly string showing what changed
// between a and b.
func humanDiff(a, b targetConfig) string {
	// FUTURE(tlim): Records like MX and SRV should have more clever output.
	// For example if only the MX priority changes, show just that.

	if a.comparableNoTTL != b.comparableNoTTL {
		// The recorddata is different:
		return fmt.Sprintf("(%s) -> (%s)", a.comparableFull, b.comparableFull)
	}

	// Just the TTLs are different:
	return fmt.Sprintf("ttl=(%d->%d) %s",
		a.rec.TTL, b.rec.TTL,
		a.comparableNoTTL)
}

var echRe = regexp.MustCompile(`ech="?([\w+/=]+)"?`)

// diffTargets is the real workhorse of the diff2 system.  All the setup has been complete,
// now we can find the differences between two zones.
func diffTargets(existing, desired []targetConfig) ChangeList {
	// fmt.Printf("DEBUG: diffTargets(\nexisting=%v\ndesired=%v\nDEBUG.\n", existing, desired)

	// Nothing to do?
	if len(existing) == 0 && len(desired) == 0 {
		return nil
	}

	echs := make(map[string]string)
	for _, v := range existing {
		matches := echRe.FindStringSubmatch(v.rec.SvcParams)
		if len(matches) == 2 {
			echs[v.rec.NameFQDN] = matches[1]
		}
	}
	for i, v := range desired {
		if strings.Contains(v.rec.SvcParams, "ech=IGNORE") {
			var unquoted, quoted string
			if _, ok := echs[v.rec.NameFQDN]; ok {
				unquoted = fmt.Sprintf("ech=%s", echs[v.rec.NameFQDN])
				quoted = fmt.Sprintf("ech=%s", echs[v.rec.NameFQDN])
			} else {
				unquoted = ""
				quoted = ""
			}
			v.rec.SvcParams = echRe.ReplaceAllString(v.rec.SvcParams, unquoted)
			v.comparableFull = echRe.ReplaceAllString(v.comparableFull, quoted)
			v.comparableNoTTL = echRe.ReplaceAllString(v.comparableNoTTL, quoted)
		}
		desired[i] = v
	}

	var instructions ChangeList

	// remove the exact matches.
	existing, desired = removeCommon(existing, desired)

	// At this point the exact matches are removed. However there may be
	// records that have the same GetTargetCombined() but different
	// TTLs.

	existing, desired, newChanges := findTTLChanges(existing, desired)
	instructions = append(instructions, newChanges...)

	// Sort by comparableFull
	sort.Slice(existing, func(i, j int) bool { return existing[i].comparableFull < existing[j].comparableFull })
	sort.Slice(desired, func(i, j int) bool { return desired[i].comparableFull < desired[j].comparableFull })

	// the remaining chunks are changes (regardless of TTL)
	mi := min(len(existing), len(desired))
	for i := range mi {
		er := existing[i].rec
		dr := desired[i].rec

		m := color.YellowString("± MODIFY %s %s %s", dr.NameFQDN, dr.Type, humanDiff(existing[i], desired[i]))

		mkc := mkChange(dr.NameFQDN, dr.Type, []string{m}, models.Records{er}, models.Records{dr})
		instructions = append(instructions, mkc)
	}

	// any left-over existing are deletes
	for i := mi; i < len(existing); i++ {
		er := existing[i].rec
		m := color.RedString("- DELETE %s %s %s", er.NameFQDN, er.Type, existing[i].comparableFull)
		instructions = append(instructions, mkDelete(er.NameFQDN, er.Type, []string{m}, models.Records{er}))
	}

	// any left-over desired are creates
	for i := mi; i < len(desired); i++ {
		dr := desired[i].rec
		m := color.GreenString("+ CREATE %s %s %s", dr.NameFQDN, dr.Type, desired[i].comparableFull)
		instructions = append(instructions, mkAdd(dr.NameFQDN, dr.Type, []string{m}, models.Records{dr}))
	}

	return instructions
}

func justMsgs(cl ChangeList) []string {
	var msgs []string
	for _, c := range cl {
		msgs = append(msgs, c.Msgs...)
	}
	return msgs
}

package normalize

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func checkRecordSetHasMultipleTTLs(records models.Records) (errs []error) {
	// The RFCs say that all records at a particular recordset should have
	// the same TTL.  Most providers don't care, and if they do the
	// dnscontrol provider code usually picks the lowest TTL for all of them.

	// General algorithm:
	// gather all records at a particular label.
	//     has[label] -> ttl -> type(s)
	// for each label, if there is more than one ttl, output ttl:A/TXT ttl:TXT/NS

	// Find the inconsistencies:
	m := make(map[string]map[uint32]map[string]bool)
	for _, r := range records {
		if !r.IsTTLSignificant() {
			continue
		}
		label := r.GetLabelFQDN()
		ttl := r.TTL
		rtype := r.Type

		if _, ok := m[label]; !ok {
			m[label] = make(map[uint32]map[string]bool)
		}
		if _, ok := m[label][ttl]; !ok {
			m[label][ttl] = make(map[string]bool)
		}
		m[label][ttl][rtype] = true
	}

	labels := make([]string, len(m))
	i := 0
	for k := range m {
		labels[i] = k
		i++
	}
	sort.Strings(labels)
	// NB(tlim): No need to de-dup labels. They come from map keys.

	for _, label := range labels {
		if len(m[label]) > 1 {
			// Invert for a more clear error message:
			r := make(map[string]map[uint32]bool)
			for ttl, rtypes := range m[label] {
				for rtype := range rtypes {
					if _, ok := r[rtype]; !ok {
						r[rtype] = make(map[uint32]bool)
					}
					r[rtype][ttl] = true
				}
			}

			// Report any cases where a RecordSet has > 1 different TTLs
			for rtype := range r {
				if len(r[rtype]) > 1 {
					result := formatInconsistency(r)
					errs = append(errs, Warning{fmt.Errorf("inconsistent TTLs at %q: %s", label, result)})
				}
			}
		}
	}

	return errs
}

func formatInconsistency(r map[string]map[uint32]bool) string {
	var rtypeResult []string
	for rtype, ttlsMap := range r {
		ttlList := make([]int, len(ttlsMap))
		i := 0
		for k := range ttlsMap {
			ttlList[i] = int(k)
			i++
		}

		sort.Ints(ttlList)

		rtypeResult = append(rtypeResult, fmt.Sprintf("%s:%v", rtype, commaSepInts(ttlList)))
	}
	sort.Strings(rtypeResult)
	return strings.Join(rtypeResult, " ")
}

func commaSepInts(list []int) string {
	slist := make([]string, len(list))
	for i, v := range list {
		slist[i] = strconv.Itoa(v)
	}
	return strings.Join(slist, ",")
}

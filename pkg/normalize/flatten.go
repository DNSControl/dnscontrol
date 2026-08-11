package normalize

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/spflib"
)

func sortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	slices.Sort(keys)
	return keys
}

// hasSpfRecords returns true if this record requests SPF unrolling.
func flattenSPFs(cfg *models.DNSConfig) []error {
	var cache spflib.CachingResolver
	var errs []error
	var err error
	for _, domain := range cfg.Domains {
		txtRecords := domain.Records.GetByType("TXT")
		// flatten all spf records that have the "flatten" metadata
		for _, txt := range txtRecords {
			var rec *spflib.SPFRecord
			txtTarget := txt.GetTargetTXTJoined()
			if txt.Metadata["flatten"] != "" || txt.Metadata["split"] != "" {
				if cache == nil {
					cache, err = spflib.NewCache("spfcache.json")
					if err != nil {
						return []error{err}
					}
				}
				rec, err = spflib.Parse(txtTarget, cache)
				if err != nil {
					errs = append(errs, err)
					continue
				}
			}
			if flatten, ok := txt.Metadata["flatten"]; ok && strings.HasPrefix(txtTarget, "v=spf1") {
				rec = rec.Flatten(flatten)
				err = txt.SetTargetTXT(rec.TXT())
				if err != nil {
					errs = append(errs, err)
					continue
				}
			}
			// now split if needed
			if split, ok := txt.Metadata["split"]; ok {
				overhead1 := 0
				// overhead1: The first segment of the SPF record
				// needs to be shorter than the others due to the overhead of
				// other (non-SPF) txt records.  If there are (for example) 50
				// bytes of txt records also on this domain record, setting
				// overhead1=50 reduces the maxLen by 50. It only affects the
				// first part of the split.
				if oh, ok := txt.Metadata["overhead1"]; ok {
					i, err := strconv.Atoi(oh)
					if err != nil {
						errs = append(errs, Warning{fmt.Errorf("split overhead1 %q is not an int", oh)})
					}
					overhead1 = i
				}

				// Default txtMaxSize will not result in multiple TXT strings
				txtMaxSize := 255
				if oh, ok := txt.Metadata["txtMaxSize"]; ok {
					i, err := strconv.Atoi(oh)
					if err != nil {
						errs = append(errs, Warning{fmt.Errorf("split txtMaxSize %q is not an int", oh)})
					}
					txtMaxSize = i
				}

				if !strings.Contains(split, "%d") {
					errs = append(errs, Warning{fmt.Errorf("split format `%s` in `%s` is not proper format (missing %%d)", split, txt.GetLabelFQDN())})
					continue
				}
				recs := rec.TXTSplit(split+"."+domain.Name, overhead1, txtMaxSize)

				for _, k := range sortedKeys(recs) {
					v := recs[k]
					if k == "@" {
						if err := txt.SetTargetTXTs(v); err != nil {
							errs = append(errs, err)
						}
					} else {
						cp, _ := txt.Copy()
						if err := cp.SetTargetTXTs(v); err != nil {
							errs = append(errs, err)
						}
						cp.SetLabelFromFQDN(k, domain.Name)
						domain.Records = append(domain.Records, cp)
					}
				}
			}
		}
	}
	if cache == nil {
		return errs
	}
	// check if cache is stale
	for _, e := range cache.ResolveErrors() {
		errs = append(errs, Warning{fmt.Errorf("problem resolving SPF record: %w", e)})
	}
	if len(cache.ResolveErrors()) == 0 {
		changed := cache.ChangedRecords()
		if len(changed) > 0 {
			if err := cache.Save("spfcache.updated.json"); err != nil {
				errs = append(errs, err)
			} else if cache.IsCachePreserved() {
				// Only warn if we loaded an existing cache file. The file is still created, which helps people enable this feature.
				errs = append(errs, Warning{fmt.Errorf("%d spf record lookups are out of date with cache (%s).\nWrote changes to spfcache.updated.json. Please rename and commit:\n    $ mv spfcache.updated.json spfcache.json\n    $ git commit -m 'Update spfcache.json' spfcache.json", len(changed), strings.Join(changed, ","))})
			}
		}
	}
	return errs
}

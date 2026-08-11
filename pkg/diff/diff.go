package diff

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/fatih/color"
)

// Correlation stores a difference between two records.
type Correlation struct {
	d        *differ
	Existing *models.RecordConfig
	Desired  *models.RecordConfig
}

// Changeset stores many Correlation.
type Changeset []Correlation

// Differ is an interface for computing the difference between two zones.
type Differ interface {
	// IncrementalDiff performs a diff on a record-by-record basis, and returns a sets for which records need to be created, deleted, or modified.
	IncrementalDiff(existing []*models.RecordConfig) (reportMsgs []string, create, toDelete, modify Changeset, actualChangeCount int, err error)
	// ChangedGroups performs a diff more appropriate for providers with a "RecordSet" model, where all records with the same name and type are grouped.
	// Individual record changes are often not useful in such scenarios. Instead we return a map of record keys to a list of change descriptions within that group.
	ChangedGroups(existing []*models.RecordConfig) (map[models.RecordKey][]string, []string, int, error)
}

type differ struct{}

// get normalized content for record. target, ttl, mxprio, and specified metadata.
func (d *differ) content(r *models.RecordConfig) string {
	return fmt.Sprintf("%s ttl=%d", r.ToComparableNoTTL(), r.TTL)
}

func (c Correlation) String() string {
	if c.Existing == nil {
		return color.GreenString("+ CREATE %s %s %s", c.Desired.Type, c.Desired.GetLabelFQDN(), c.d.content(c.Desired))
	}
	if c.Desired == nil {
		return color.RedString("- DELETE %s %s %s", c.Existing.Type, c.Existing.GetLabelFQDN(), c.d.content(c.Existing))
	}
	return color.YellowString("± MODIFY %s %s: (%s) -> (%s)", c.Existing.Type, c.Existing.GetLabelFQDN(), c.d.content(c.Existing), c.d.content(c.Desired))
}

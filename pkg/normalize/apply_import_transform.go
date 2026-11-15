package normalize

import (
	"fmt"
	"slices"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nameutil"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
	"github.com/DNSControl/dnscontrol/v5/pkg/transform"
)

func transformCNAME(target, oldDomain, newDomain, suffixstrip string) string {
	// Canonicalize the target.  Add the newDomain minus the suffixstrip.
	//  foo -> foo.oldDomain.newDomain
	//  foo. -> foo.newDomain
	nd := strings.TrimPrefix(newDomain, suffixstrip+".")
	if strings.HasSuffix(target, ".") {
		return target + nd + "."
	}
	return nameutil.ToFqdnWithDot(target, oldDomain) + nd + "."
}

func newRec(rec *models.RecordConfig, ttl uint32) *models.RecordConfig {
	rec2, _ := rec.Copy()
	if ttl != 0 {
		rec2.TTL = ttl
	}
	return rec2
}

func transformLabel(label, suffixstrip string) (string, error) {
	if suffixstrip == "" {
		return label, nil
	}
	suffixstrip = "." + suffixstrip
	if !strings.HasSuffix(label, suffixstrip) {
		return "", fmt.Errorf("label %q does not end with %q", label, suffixstrip)
	}
	return label[:len(label)-len(suffixstrip)], nil
}

// import_transform imports the records of one zone into another, modifying records along the way.
func importTransform(srcDomain, dstDomain *models.DomainConfig,
	transforms []transform.IPConversion, ttl uint32, suffixstrip string,
) error {
	// Read srcDomain.Records, transform, and append to dstDomain.Records:
	// 1. Skip any that aren't A or CNAMEs.
	// 2. Append destDomainname to the end of the label.
	// 3. For CNAMEs, append destDomainname to the end of the target.
	// 4. For As, change the target as described the transforms.

	for _, rec := range srcDomain.Records {
		// If this record is marked to be skipped, skip it.
		if rec.Metadata["import_transform_skip"] != "" {
			continue
		}
		// If the dstDomain already has a record with this type+label, skip it.
		if dstDomain.Records.HasRecordTypeName(rec.Type, rec.GetLabelFQDN()) {
			continue
		}
		switch rec.Type {
		case "A":
			addr := rec.AsA().Addr
			trs, err := transform.IPToList(addr, transforms)
			if err != nil {
				return fmt.Errorf("import_transform: TransformIP(%v, %v) returned err=%w", addr, transforms, err)
			}
			for _, tr := range trs {
				r := newRec(rec, ttl)
				l, err := transformLabel(r.GetLabelFQDN(), suffixstrip)
				if err != nil {
					return err
				}
				r.SetLabel(l, dstDomain.Name)

				rd, err := models.MakeA(dstDomain.Name, rec.Metadata, nrc.Flags{}, tr.String())
				if err != nil {
					return err
				}
				r.SetRDATA(rd)

				dstDomain.Records = append(dstDomain.Records, r)
			}
		case "CNAME":
			r := newRec(rec, ttl)
			l, err := transformLabel(r.GetLabelFQDN(), suffixstrip)
			if err != nil {
				return err
			}
			r.SetLabel(l, dstDomain.Name)

			rd, err := models.MakeCNAME(dstDomain.Name, rec.Metadata, nrc.Flags{}, transformCNAME(r.AsCNAME().Target, srcDomain.Name, dstDomain.Name, suffixstrip))
			if err != nil {
				return err
			}
			r.SetRDATA(rd)

			dstDomain.Records = append(dstDomain.Records, r)
		default:
			// Anything else is ignored.
			continue
		}
	}
	return nil
}

// deleteImportTransformRecords deletes any IMPORT_TRANSFORM records from a domain.
func deleteImportTransformRecords(domain *models.DomainConfig) {
	for i, rec := range slices.Backward(domain.Records) {

		if rec.Type == "IMPORT_TRANSFORM" {
			domain.Records = append(domain.Records[:i], domain.Records[i+1:]...)
		}
	}
}

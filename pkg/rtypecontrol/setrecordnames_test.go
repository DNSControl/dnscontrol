package rtypecontrol

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
)

func TestSetRecordNames(t *testing.T) {
	dc := &domaintags.DomainNameVarieties{
		NameRaw:     "example.com",
		NameASCII:   "example.com",
		NameUnicode: "example.com",
	}
	dcIDN := &domaintags.DomainNameVarieties{
		NameRaw:     "bücher.com",
		NameASCII:   "xn--bcher-kva.com",
		NameUnicode: "bücher.com",
	}

	tests := []struct {
		name        string
		rec         *models.RecordConfig
		dc          *domaintags.DomainNameVarieties
		n           string
		expectedErr bool
		expectedRec *models.RecordConfig
	}{

		{
			name: "normal_at",
			rec:  &models.RecordConfig{},
			dc:   dc,
			n:    "@",
			expectedRec: &models.RecordConfig{
				NameRaw:         "@",
				Name:            "@",
				NameUnicode:     "@",
				NameFQDNRaw:     "example.com",
				NameFQDN:        "example.com",
				NameFQDNUnicode: "example.com",
			},
		},
		{
			name: "normal_label",
			rec:  &models.RecordConfig{},
			dc:   dc,
			n:    "www",
			expectedRec: &models.RecordConfig{
				NameRaw:         "www",
				Name:            "www",
				NameUnicode:     "www",
				NameFQDNRaw:     "www.example.com",
				NameFQDN:        "www.example.com",
				NameFQDNUnicode: "www.example.com",
			},
		},
		{
			name: "normal_idn_label",
			rec:  &models.RecordConfig{},
			dc:   dc,
			n:    "bücher",
			expectedRec: &models.RecordConfig{
				NameRaw:         "bücher",
				Name:            "xn--bcher-kva",
				NameUnicode:     "bücher",
				NameFQDNRaw:     "bücher.example.com",
				NameFQDN:        "xn--bcher-kva.example.com",
				NameFQDNUnicode: "bücher.example.com",
			},
		},
		{
			name: "normal_idn_domain",
			rec:  &models.RecordConfig{},
			dc:   dcIDN,
			n:    "www",
			expectedRec: &models.RecordConfig{
				NameRaw:         "www",
				Name:            "www",
				NameUnicode:     "www",
				NameFQDNRaw:     "www.bücher.com",
				NameFQDN:        "www.xn--bcher-kva.com",
				NameFQDNUnicode: "www.bücher.com",
			},
		},
		{
			name: "extend_at",
			rec:  &models.RecordConfig{SubDomain: "sub"},
			dc:   dc,
			n:    "@",
			expectedRec: &models.RecordConfig{
				SubDomain:       "sub",
				NameRaw:         "sub",
				Name:            "sub",
				NameUnicode:     "sub",
				NameFQDNRaw:     "sub.example.com",
				NameFQDN:        "sub.example.com",
				NameFQDNUnicode: "sub.example.com",
			},
		},
		{
			name: "extend_label",
			rec:  &models.RecordConfig{SubDomain: "sub"},
			dc:   dc,
			n:    "www",
			expectedRec: &models.RecordConfig{
				SubDomain:       "sub",
				NameRaw:         "www.sub",
				Name:            "www.sub",
				NameUnicode:     "www.sub",
				NameFQDNRaw:     "www.sub.example.com",
				NameFQDN:        "www.sub.example.com",
				NameFQDNUnicode: "www.sub.example.com",
			},
		},
		{
			name: "extend_idn_subdomain",
			rec:  &models.RecordConfig{SubDomain: "bücher"},
			dc:   dc,
			n:    "www",
			expectedRec: &models.RecordConfig{
				SubDomain:       "bücher",
				NameRaw:         "www.bücher",
				Name:            "www.xn--bcher-kva",
				NameUnicode:     "www.bücher",
				NameFQDNRaw:     "www.bücher.example.com",
				NameFQDN:        "www.xn--bcher-kva.example.com",
				NameFQDNUnicode: "www.bücher.example.com",
			},
		},
		{
			name: "extend_idn_label",
			rec:  &models.RecordConfig{SubDomain: "sub"},
			dc:   dc,
			n:    "bücher",
			expectedRec: &models.RecordConfig{
				SubDomain:       "sub",
				NameRaw:         "bücher.sub",
				Name:            "xn--bcher-kva.sub",
				NameUnicode:     "bücher.sub",
				NameFQDNRaw:     "bücher.sub.example.com",
				NameFQDN:        "xn--bcher-kva.sub.example.com",
				NameFQDNUnicode: "bücher.sub.example.com",
			},
		},
		{
			name: "extend_idn_subdomain_and_label",
			rec:  &models.RecordConfig{SubDomain: "bücher"},
			dc:   dc,
			n:    "könig",
			expectedRec: &models.RecordConfig{
				SubDomain:       "bücher",
				NameRaw:         "könig.bücher",
				Name:            "xn--knig-5qa.xn--bcher-kva",
				NameUnicode:     "könig.bücher",
				NameFQDNRaw:     "könig.bücher.example.com",
				NameFQDN:        "xn--knig-5qa.xn--bcher-kva.example.com",
				NameFQDNUnicode: "könig.bücher.example.com",
			},
		},
		{
			name: "extend_idn_domain_and_subdomain",
			rec:  &models.RecordConfig{SubDomain: "bücher"},
			dc:   dcIDN,
			n:    "www",
			expectedRec: &models.RecordConfig{
				SubDomain:       "bücher",
				NameRaw:         "www.bücher",
				Name:            "www.xn--bcher-kva",
				NameUnicode:     "www.bücher",
				NameFQDNRaw:     "www.bücher.bücher.com",
				NameFQDN:        "www.xn--bcher-kva.xn--bcher-kva.com",
				NameFQDNUnicode: "www.bücher.bücher.com",
			},
		},

		{
			name: "dotted_normal_at",
			rec:  &models.RecordConfig{},
			dc:   dc,
			n:    "example.com.",
			expectedRec: &models.RecordConfig{
				Name:            "@",
				NameRaw:         "@",
				NameUnicode:     "@",
				NameFQDN:        "example.com",
				NameFQDNRaw:     "example.com",
				NameFQDNUnicode: "example.com",
			},
		},
		{
			name: "dotted_normal_label_outside",
			rec:  &models.RecordConfig{},
			dc:   dc,
			n:    "www.example.com.",
			expectedRec: &models.RecordConfig{
				Name:            "www",
				NameRaw:         "www",
				NameUnicode:     "www",
				NameFQDN:        "www.example.com",
				NameFQDNRaw:     "www.example.com",
				NameFQDNUnicode: "www.example.com",
			},
		},
		{
			name: "dotted_normal_idn_label",
			rec:  &models.RecordConfig{},
			dc:   dc,
			n:    "bücher.example.com.",
			expectedRec: &models.RecordConfig{
				Name:            "xn--bcher-kva",
				NameRaw:         "bücher",
				NameUnicode:     "bücher",
				NameFQDN:        "xn--bcher-kva.example.com",
				NameFQDNRaw:     "bücher.example.com",
				NameFQDNUnicode: "bücher.example.com",
			},
		},
		{
			name: "dotted_normal_idn_domain",
			rec:  &models.RecordConfig{},
			dc:   dcIDN,
			n:    "www.bücher.com.",
			expectedRec: &models.RecordConfig{
				Name:            "www",
				NameRaw:         "www",
				NameUnicode:     "www",
				NameFQDN:        "www.xn--bcher-kva.com",
				NameFQDNRaw:     "www.bücher.com",
				NameFQDNUnicode: "www.bücher.com",
			},
		},
		{
			name:        "dotted_extend_at",
			rec:         &models.RecordConfig{SubDomain: "sub"},
			dc:          dc,
			n:           "example.com.",
			expectedErr: true,
		},
		{
			name:        "dotted_extend_label",
			rec:         &models.RecordConfig{SubDomain: "sub"},
			dc:          dc,
			n:           "www.example.com.",
			expectedErr: true,
		},
		{
			name: "dotted_extend_idn_subdomain",
			rec:  &models.RecordConfig{SubDomain: "bücher"},
			dc:   dc,
			n:    "www.bücher.example.com.",
			expectedRec: &models.RecordConfig{
				SubDomain: "bücher",

				NameRaw:         "www.bücher",
				Name:            "www.xn--bcher-kva",
				NameUnicode:     "www.bücher",
				NameFQDNRaw:     "www.bücher.example.com",
				NameFQDN:        "www.xn--bcher-kva.example.com",
				NameFQDNUnicode: "www.bücher.example.com",
			},
		},
		{
			name: "dotted_extend_idn_label",
			rec:  &models.RecordConfig{SubDomain: "sub"},
			dc:   dc,
			n:    "bücher.sub.example.com.",
			expectedRec: &models.RecordConfig{
				SubDomain:       "sub",
				Name:            "xn--bcher-kva.sub",
				NameRaw:         "bücher.sub",
				NameUnicode:     "bücher.sub",
				NameFQDN:        "xn--bcher-kva.sub.example.com",
				NameFQDNRaw:     "bücher.sub.example.com",
				NameFQDNUnicode: "bücher.sub.example.com",
			},
		},
		{
			name: "dotted_extend_idn_subdomain_and_label",
			rec:  &models.RecordConfig{SubDomain: "bücher"},
			dc:   dc,
			n:    "könig.bücher.example.com.",
			expectedRec: &models.RecordConfig{
				SubDomain:       "bücher",
				Name:            "xn--knig-5qa.xn--bcher-kva",
				NameRaw:         "könig.bücher",
				NameUnicode:     "könig.bücher",
				NameFQDN:        "xn--knig-5qa.xn--bcher-kva.example.com",
				NameFQDNRaw:     "könig.bücher.example.com",
				NameFQDNUnicode: "könig.bücher.example.com",
			},
		},
		{
			name: "dotted_extend_idn_domain_and_subdomain",
			rec:  &models.RecordConfig{SubDomain: "bücher"},
			dc:   dcIDN,
			n:    "www.bücher.bücher.com.",
			expectedRec: &models.RecordConfig{
				SubDomain:       "bücher",
				Name:            "www.xn--bcher-kva",
				NameRaw:         "www.bücher",
				NameUnicode:     "www.bücher",
				NameFQDN:        "www.xn--bcher-kva.xn--bcher-kva.com",
				NameFQDNRaw:     "www.bücher.bücher.com",
				NameFQDNUnicode: "www.bücher.bücher.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := setRecordNames(tt.rec, tt.dc, tt.n)
			if (gotErr != nil && (!tt.expectedErr)) || (gotErr == nil && tt.expectedErr) {
				t.Errorf("Error: got \"%v\", want %v", gotErr, tt.expectedErr)
			} else if gotErr != nil && tt.expectedErr {
				// Expected error, test passed.
			} else {
				if tt.rec.NameRaw != tt.expectedRec.NameRaw {
					t.Errorf("NameRaw: got %q, want %q", tt.rec.NameRaw, tt.expectedRec.NameRaw)
				}
				if tt.rec.Name != tt.expectedRec.Name {
					t.Errorf("Name: got %q, want %q", tt.rec.Name, tt.expectedRec.Name)
				}
				if tt.rec.NameUnicode != tt.expectedRec.NameUnicode {
					t.Errorf("NameUnicode: got %q, want %q", tt.rec.NameUnicode, tt.expectedRec.NameUnicode)
				}
				if tt.rec.NameFQDNRaw != tt.expectedRec.NameFQDNRaw {
					t.Errorf("NameFQDNRaw: got %q, want %q", tt.rec.NameFQDNRaw, tt.expectedRec.NameFQDNRaw)
				}
				if tt.rec.NameFQDN != tt.expectedRec.NameFQDN {
					t.Errorf("NameFQDN: got %q, want %q", tt.rec.NameFQDN, tt.expectedRec.NameFQDN)
				}
				if tt.rec.NameFQDNUnicode != tt.expectedRec.NameFQDNUnicode {
					t.Errorf("NameFQDNUnicode: got %q, want %q", tt.rec.NameFQDNUnicode, tt.expectedRec.NameFQDNUnicode)
				}
			}
		})
	}
}

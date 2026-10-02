package providers

import "testing"

func TestDefinitionDocumentationURLs(t *testing.T) {
	for _, tc := range []struct {
		name, override, want string
	}{
		{"MiXeD_TYPE", "", "https://docs.dnscontrol.org/provider/mixed_type"},
		{"RENAMED", "https://docs.dnscontrol.org/provider/legacy", "https://docs.dnscontrol.org/provider/legacy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateDefinitions(t)
			Register[*None](tc.name, Definition{
				FriendlyName: "An unrelated brand name", Aliases: []string{"OLD_NAME"},
				DocsURL: tc.override,
			})
			for _, name := range []string{tc.name, "OLD_NAME"} {
				def, _ := GetDefinition(name)
				if def.DocsURL != tc.want {
					t.Fatalf("%s: documentation URL = %q; want %q", name, def.DocsURL, tc.want)
				}
			}
		})
	}
}

func TestRegisterRejectsRedundantDocsURL(t *testing.T) {
	isolateDefinitions(t)
	assertRegistrationPanics(t, "DocsURL override matches derived URL", func() {
		Register[*None]("MiXeD_TYPE", Definition{
			FriendlyName: "Example", DocsURL: "https://docs.dnscontrol.org/provider/mixed_type",
		})
	})
	if len(definitions) != 0 {
		t.Fatal("invalid registration was partially published")
	}
}

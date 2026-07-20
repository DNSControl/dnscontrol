package cfsingleredirect

import (
	"regexp"
	"testing"

	"github.com/gobwas/glob"
)

func Test_makeSingleDirectRule(t *testing.T) {
	tests := []struct {
		name string
		//
		pattern string
		replace string
		//
		wantMatch string
		wantExpr  string
		wantErr   bool
	}{
		{
			name:      "000",
			pattern:   "example.com/",
			replace:   "foo.com",
			wantMatch: `http.host eq "example.com" and http.request.uri.path eq "/"`,
			wantExpr:  `concat("https://foo.com", "")`,
			wantErr:   false,
		},

		/*
			All the test-cases I could find in dnsconfig.js

			Generated with this:

			dnscontrol print-ir --pretty |grep '"target' |grep , | sed -e 's@"target":@@g' > /tmp/list
			vim /tmp/list    # removed the obvious duplicates
			awk < /tmp/list -v q='"' -F, '{ print "{" ; print "name: " q NR  q ","  ; print "pattern: " $1 q "," ; print "replace: " q $2 "," ; print "wantMatch: `FIXME`," ; print "wantExpr:  `FIXME`," ; print "wantErr:   false," ; print "},"  }' | pbcopy

		*/

		{
			name:      "1",
			pattern:   "https://i-dev.sstatic.net/",
			replace:   "https://stackexchange.com/",
			wantMatch: `http.host eq "i-dev.sstatic.net" and http.request.uri.path eq "/"`,
			wantExpr:  `concat("https://stackexchange.com/", "")`,
			wantErr:   false,
		},
		{
			name:      "2",
			pattern:   "https://i.stack.imgur.com/*",
			replace:   "https://i.sstatic.net/$1",
			wantMatch: `http.host eq "i.stack.imgur.com"`,
			wantExpr:  `concat("https://i.sstatic.net", http.request.uri.path)`,
			wantErr:   false,
		},
		{
			name:      "3",
			pattern:   "https://img.stack.imgur.com/*",
			replace:   "https://i.sstatic.net/$1",
			wantMatch: `http.host eq "img.stack.imgur.com"`,
			wantExpr:  `concat("https://i.sstatic.net", http.request.uri.path)`,
			wantErr:   false,
		},
		{
			name:      "4",
			pattern:   "https://insights.stackoverflow.com/",
			replace:   "https://survey.stackoverflow.co",
			wantMatch: `http.host eq "insights.stackoverflow.com" and http.request.uri.path eq "/"`,
			wantExpr:  `concat("https://survey.stackoverflow.co", "")`,
			wantErr:   false,
		},
		{
			name:      "5",
			pattern:   "https://insights.stackoverflow.com/trends",
			replace:   "https://trends.stackoverflow.co",
			wantMatch: `http.host eq "insights.stackoverflow.com" and http.request.uri.path eq "/trends"`,
			wantExpr:  `concat("https://trends.stackoverflow.co", "")`,
			wantErr:   false,
		},
		{
			name:      "6",
			pattern:   "https://insights.stackoverflow.com/trends/",
			replace:   "https://trends.stackoverflow.co",
			wantMatch: `http.host eq "insights.stackoverflow.com" and http.request.uri.path eq "/trends/"`,
			wantExpr:  `concat("https://trends.stackoverflow.co", "")`,
			wantErr:   false,
		},
		{
			name:      "7",
			pattern:   "https://insights.stackoverflow.com/survey/2021",
			replace:   "https://survey.stackoverflow.co/2021",
			wantMatch: `http.host eq "insights.stackoverflow.com" and http.request.uri.path eq "/survey/2021"`,
			wantExpr:  `concat("https://survey.stackoverflow.co/2021", "")`,
			wantErr:   false,
		},
		{
			name:      "28",
			pattern:   "*stackoverflow.help/support/solutions/articles/36000241656-write-an-article",
			replace:   "https://stackoverflow.help/en/articles/4397209-write-an-article",
			wantMatch: `( http.host eq "stackoverflow.help" or ends_with(http.host, ".stackoverflow.help") ) and http.request.uri.path eq "/support/solutions/articles/36000241656-write-an-article"`,
			wantExpr:  `concat("https://stackoverflow.help/en/articles/4397209-write-an-article", "")`,
			wantErr:   false,
		},
		{
			name:      "29",
			pattern:   "*stackoverflow.careers/*",
			replace:   "https://careers.stackoverflow.com/$2",
			wantMatch: `http.host eq "stackoverflow.careers" or ends_with(http.host, ".stackoverflow.careers")`,
			wantExpr:  `concat("https://careers.stackoverflow.com", http.request.uri.path)`,
			wantErr:   false,
		},
		{
			name:      "31",
			pattern:   "stackenterprise.com/*",
			replace:   "https://stackoverflow.co/teams/",
			wantMatch: `http.host eq "stackenterprise.com"`,
			wantExpr:  `concat("https://stackoverflow.co/teams/", "")`,
			wantErr:   false,
		},
		{
			name:      "33",
			pattern:   "meta.*yodeya.com/*",
			replace:   "https://judaism.meta.stackexchange.com/$2",
			wantMatch: `http.host matches r###"^meta\..*yodeya\.com$"###`,
			wantExpr:  `concat("https://judaism.meta.stackexchange.com", http.request.uri.path)`,
			wantErr:   false,
		},
		{
			name:      "34",
			pattern:   "chat.*yodeya.com/*",
			replace:   "https://chat.stackexchange.com/?tab=site\u0026host=judaism.stackexchange.com",
			wantMatch: `http.host matches r###"^chat\..*yodeya\.com$"###`,
			wantExpr:  `concat("https://chat.stackexchange.com/?tab=site&host=judaism.stackexchange.com", "")`,
			wantErr:   false,
		},
		{
			name:      "35",
			pattern:   "*yodeya.com/*",
			replace:   "https://judaism.stackexchange.com/$2",
			wantMatch: `http.host eq "yodeya.com" or ends_with(http.host, ".yodeya.com")`,
			wantExpr:  `concat("https://judaism.stackexchange.com", http.request.uri.path)`,
			wantErr:   false,
		},
		{
			name:      "36",
			pattern:   "meta.*seasonedadvice.com/*",
			replace:   "https://cooking.meta.stackexchange.com/$2",
			wantMatch: `http.host matches r###"^meta\..*seasonedadvice\.com$"###`,
			wantExpr:  `concat("https://cooking.meta.stackexchange.com", http.request.uri.path)`,
			wantErr:   false,
		},
		{
			name:      "70",
			pattern:   "collectivesonstackoverflow.co/*",
			replace:   "https://stackoverflow.com/collectives-on-stack-overflow",
			wantMatch: `http.host eq "collectivesonstackoverflow.co"`,
			wantExpr:  `concat("https://stackoverflow.com/collectives-on-stack-overflow", "")`,
			wantErr:   false,
		},
		{
			name:      "71",
			pattern:   "*collectivesonstackoverflow.co/*",
			replace:   "https://stackoverflow.com/collectives-on-stack-overflow",
			wantMatch: `http.host eq "collectivesonstackoverflow.co" or ends_with(http.host, ".collectivesonstackoverflow.co")`,
			wantExpr:  `concat("https://stackoverflow.com/collectives-on-stack-overflow", "")`,
			wantErr:   false,
		},
		{
			name:      "76",
			pattern:   "*stackexchange.ca/*",
			replace:   "https://stackexchange.com/$2",
			wantMatch: `http.host eq "stackexchange.ca" or ends_with(http.host, ".stackexchange.ca")`,
			wantExpr:  `concat("https://stackexchange.com", http.request.uri.path)`,
			wantErr:   false,
		},

		// https://github.com/DNSControl/dnscontrol/issues/2313#issuecomment-2197296025
		{
			name:      "pro-sumer1",
			pattern:   "domain.tld/.well-known*",
			replace:   "https://social.domain.tld/.well-known$1",
			wantMatch: `(starts_with(http.request.uri.path, "/.well-known") and http.host eq "domain.tld")`,
			wantExpr:  `concat("https://social.domain.tld", http.request.uri.path)`,
			wantErr:   false,
		},
		{
			name:      "pro-sumer2",
			pattern:   "domain.tld/users*",
			replace:   "https://social.domain.tld/users$1",
			wantMatch: `(starts_with(http.request.uri.path, "/users") and http.host eq "domain.tld")`,
			wantExpr:  `concat("https://social.domain.tld", http.request.uri.path)`,
			wantErr:   false,
		},
		{
			name:      "pro-sumer3",
			pattern:   "domain.tld/@*",
			replace:   `https://social.domain.tld/@$1`,
			wantMatch: `(starts_with(http.request.uri.path, "/@") and http.host eq "domain.tld")`,
			wantExpr:  `concat("https://social.domain.tld", http.request.uri.path)`,
			wantErr:   false,
		},

		{
			name:      "stackentwild",
			pattern:   "*stackoverflowenterprise.com/*",
			replace:   "https://www.stackoverflowbusiness.com/enterprise/$2",
			wantMatch: `http.host eq "stackoverflowenterprise.com" or ends_with(http.host, ".stackoverflowenterprise.com")`,
			wantExpr:  `concat("https://www.stackoverflowbusiness.com", "/enterprise", http.request.uri.path)`,
			wantErr:   false,
		},

		//
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMatch, gotExpr, err := makeRuleFromPattern(tt.pattern, tt.replace)
			if (err != nil) != tt.wantErr {
				t.Errorf("makeSingleDirectRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotMatch != tt.wantMatch {
				t.Errorf("makeSingleDirectRule() MATCH = %v\n                                                  want %v", gotMatch, tt.wantMatch)
			}
			if gotExpr != tt.wantExpr {
				t.Errorf("makeSingleDirectRule()  EXPR = %v\n                                                  want %v", gotExpr, tt.wantExpr)
			}
			// _ = gotType
		})
	}
}

func Test_normalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    string
		want1   string
		want2   string
		wantErr bool
	}{
		{
			s:     "foo.com",
			want:  "https://foo.com",
			want1: "foo.com",
			want2: "",
		},
		{
			s:     "http://foo.com",
			want:  "https://foo.com",
			want1: "foo.com",
			want2: "",
		},
		{
			s:     "https://foo.com",
			want:  "https://foo.com",
			want1: "foo.com",
			want2: "",
		},

		{
			s:     "foo.com/bar",
			want:  "https://foo.com/bar",
			want1: "foo.com",
			want2: "/bar",
		},
		{
			s:     "http://foo.com/bar",
			want:  "https://foo.com/bar",
			want1: "foo.com",
			want2: "/bar",
		},
		{
			s:     "https://foo.com/bar",
			want:  "https://foo.com/bar",
			want1: "foo.com",
			want2: "/bar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, err := normalizeURL(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("normalizeURL() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("normalizeURL() got1 = %v, want1 %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("normalizeURL() got2 = %v, want2 %v", got2, tt.want2)
			}
		})
	}
}

func Test_simpleGlobToRegex(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"1", `foo`, `^foo$`},
		{"2", `fo.o`, `^fo\.o$`},
		{"3", `*foo`, `.*foo$`},
		{"4", `foo*`, `^foo.*`},
		{"5", `f.oo*`, `^f\.oo.*`},
		{"6", `f*oo*`, `^f.*oo.*`},
	}

	data := []string{
		"bar",
		"foo",
		"foofoo",
		"ONEfooTWO",
		"fo",
		"frankfodog",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := simpleGlobToRegex(tt.pattern)
			if got != tt.want {
				t.Errorf("simpleGlobToRegex() = %v, want %v", got, tt.want)
			}

			// Make sure the regex compiles and gets the same result when matching against strings in data.
			for i, d := range data {
				rm, err := regexp.MatchString(got, d)
				if err != nil {
					t.Errorf("simpleGlobToRegex() = %003d  can not compile: %v", i, err)
				}

				g := glob.MustCompile(tt.pattern)
				gm := g.Match(d) // true

				if gm != rm {
					t.Errorf("simpleGlobToRegex() = %003d  glob: %v '%v'  regexp: %v '%v'", i, gm, tt.pattern, rm, got)
				}
			}
		})
	}
}

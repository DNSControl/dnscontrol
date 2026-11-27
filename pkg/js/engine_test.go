package js

// These tests pin down the behavior of the JavaScript runtime itself
// (event loop, timers, Promises, fetch, underscore, and the Go builtins'
// argument checking) so that replacing the engine (otto → sobek, see
// https://github.com/DNSControl/dnscontrol/issues/5135) can be verified.
//
// Error checks match only the message text, not the "Error: " prefix or
// any position information, since those depend on the engine.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	testifyrequire "github.com/stretchr/testify/require"
)

// runTXT runs script and returns the joined TXT target of the first
// record of the first domain. Scripts report results by calling
// D("x.com","none",TXT("@", result)), often from an async callback.
func runTXT(t *testing.T, script string) string {
	t.Helper()
	resetDirectories(t)
	conf, err := ExecuteJavascriptString([]byte(script), true, nil)
	testifyrequire.NoError(t, err)
	testifyrequire.NotEmpty(t, conf.Domains, "script did not create a domain")
	testifyrequire.NotEmpty(t, conf.Domains[0].Records, "script did not create a record")
	return conf.Domains[0].Records[0].GetTargetTXTJoined()
}

// resetDirectories makes require() and glob() resolve paths relative to
// the working directory (the repo root). ExecuteJavascriptString does not
// reset these package globals, so otherwise they keep whatever value an
// earlier ExecuteJavaScript call left behind.
func resetDirectories(t *testing.T) {
	t.Helper()
	oldCur, oldCfg := currentDirectory, configDirectory
	currentDirectory, configDirectory = ".", ""
	t.Cleanup(func() { currentDirectory, configDirectory = oldCur, oldCfg })
}

// skipClearTimeoutUnderRace skips tests that call clearTimeout() when the
// race detector is on. ottoext's Loop.Remove calls Loop.Ready from a new
// goroutine, which reads Loop.closed without a lock while Loop.Run writes
// it (and can send on the closed channel). This goes away with the move
// to sobek: https://github.com/DNSControl/dnscontrol/issues/5135
func skipClearTimeoutUnderRace(t *testing.T) {
	t.Helper()
	if raceEnabled {
		t.Skip("ottoext's clearTimeout has a data race; see issue 5135")
	}
}

// enableFetch turns on fetch() for the duration of the test.
func enableFetch(t *testing.T) {
	t.Helper()
	old := EnableFetch
	EnableFetch = true
	t.Cleanup(func() { EnableFetch = old })
}

func TestTimers(t *testing.T) {
	tests := []struct{ desc, script, want string }{
		{
			"setTimeout runs in delay order",
			`var o = [];
			setTimeout(function () { o.push("b"); }, 20);
			setTimeout(function () { o.push("a"); }, 0);
			setTimeout(function () { D("x.com", "none", TXT("@", o.join(","))); }, 50);`,
			"a,b",
		},
		{
			"setTimeout passes no surprises to the script after it",
			`var o = ["sync"];
			setTimeout(function () { o.push("async"); D("x.com", "none", TXT("@", o.join(","))); }, 0);
			o.push("sync2");`,
			"sync,sync2,async",
		},
		{
			"clearTimeout cancels",
			`var c = setTimeout(function () { D("y.com", "none"); }, 1);
			clearTimeout(c);
			D("x.com", "none", TXT("@", "ok"));`,
			"ok",
		},
		{
			"setInterval repeats until clearInterval",
			`var n = 0;
			var id = setInterval(function () {
				n++;
				if (n == 3) { clearInterval(id); D("x.com", "none", TXT("@", "" + n)); }
			}, 1);`,
			"3",
		},
	}
	for _, tst := range tests {
		t.Run(tst.desc, func(t *testing.T) {
			if strings.Contains(tst.script, "clear") {
				skipClearTimeoutUnderRace(t)
			}
			testifyrequire.Equal(t, tst.want, runTXT(t, tst.script))
		})
	}
}

func TestClearTimeoutLeavesNoDomain(t *testing.T) {
	skipClearTimeoutUnderRace(t)
	conf, err := ExecuteJavascriptString([]byte(`
		var c = setTimeout(function () { D("y.com", "none"); }, 1);
		clearTimeout(c);`), true, nil)
	testifyrequire.NoError(t, err)
	testifyrequire.Empty(t, conf.Domains)
}

func TestPromises(t *testing.T) {
	tests := []struct{ desc, script, want string }{
		{
			"then chain",
			`Promise.resolve(1)
				.then(function (v) { return v + 1; })
				.then(function (v) { D("x.com", "none", TXT("@", "" + v)); });`,
			"2",
		},
		{
			"Promise.all with a timer",
			`Promise.all([
				Promise.resolve(1),
				new Promise(function (resolve) { setTimeout(function () { resolve(2); }, 1); })
			]).then(function (v) { D("x.com", "none", TXT("@", v.join(","))); });`,
			"1,2",
		},
		{
			"catch receives the rejection",
			`new Promise(function (resolve, reject) { reject("r"); })
				.catch(function (e) { D("x.com", "none", TXT("@", "caught " + e)); });`,
			"caught r",
		},
		{
			"error thrown in then reaches catch",
			`Promise.resolve()
				.then(function () { throw new Error("thrown"); })
				.catch(function (e) { D("x.com", "none", TXT("@", e.message)); });`,
			"thrown",
		},
		{
			"then callbacks run after the synchronous script",
			`var o = [];
			Promise.resolve().then(function () { o.push("then"); D("x.com", "none", TXT("@", o.join(","))); });
			o.push("sync");`,
			"sync,then",
		},
		// Current behavior: an unhandled rejection is silently ignored.
		// The engine migration may deliberately turn this into an error;
		// if so, update these cases and add a release note.
		{
			"unhandled rejection is ignored",
			`Promise.reject(new Error("boom")); D("x.com", "none", TXT("@", "ok"));`,
			"ok",
		},
		{
			"unhandled throw in then is ignored",
			`Promise.resolve().then(function () { throw new Error("boom"); });
			D("x.com", "none", TXT("@", "ok"));`,
			"ok",
		},
	}
	for _, tst := range tests {
		t.Run(tst.desc, func(t *testing.T) {
			testifyrequire.Equal(t, tst.want, runTXT(t, tst.script))
		})
	}
}

// fetchServer echoes the request and sets some response headers.
func fetchServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Test", "yes")
		w.Header().Add("X-Multi", "1")
		w.Header().Add("X-Multi", "2")
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
		}
		fmt.Fprintf(w, `{"method":%q,"path":%q,"hdr":%q,"body":%q}`,
			r.Method, r.URL.Path, r.Header.Get("X-Req"), string(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetch(t *testing.T) {
	enableFetch(t)
	srv := fetchServer(t)

	tests := []struct{ desc, script, want string }{
		{
			"json() of a GET",
			`FETCH("URL/a")
				.then(function (r) { return r.json(); })
				.then(function (j) { D("x.com", "none", TXT("@", j.method + " " + j.path)); });`,
			"GET /a",
		},
		{
			"lowercase fetch() is also available",
			`fetch("URL/b")
				.then(function (r) { return r.json(); })
				.then(function (j) { D("x.com", "none", TXT("@", j.path)); });`,
			"/b",
		},
		{
			"POST with headers and body",
			`FETCH("URL/p", { method: "POST", headers: { "X-Req": "hi" }, body: "payload" })
				.then(function (r) { return r.json(); })
				.then(function (j) { D("x.com", "none", TXT("@", [j.method, j.hdr, j.body].join("|"))); });`,
			"POST|hi|payload",
		},
		{
			// statusText is "200 OK" (Go's res.Status), not the "OK" that
			// the fetch standard specifies. type is empty.
			"response fields",
			`FETCH("URL/r").then(function (r) {
				D("x.com", "none", TXT("@", [r.ok, r.status, r.statusText, r.type].join("|")));
			});`,
			"true|200|200 OK|",
		},
		{
			"response headers are case-insensitive and multi-valued",
			`FETCH("URL/h").then(function (r) {
				D("x.com", "none", TXT("@", [
					r.headers.get("x-test"), r.headers.get("X-Test"), r.headers.has("X-TEST"),
					r.headers.has("X-Absent"), r.headers.getAll("X-Multi").join(",")
				].join("|")));
			});`,
			"yes|yes|true|false|1,2",
		},
		{
			"text() returns the body",
			`FETCH("URL/t")
				.then(function (r) { return r.text(); })
				.then(function (s) { D("x.com", "none", TXT("@", s)); });`,
			`{"method":"GET","path":"/t","hdr":"","body":""}`,
		},
		{
			// HTTP errors resolve (they do not reject). ok is true even
			// for a 404, unlike the fetch standard (200-299 only).
			"HTTP errors resolve, ok is always true",
			`FETCH("URL/missing").then(function (r) {
				D("x.com", "none", TXT("@", [r.ok, r.status].join("|")));
			});`,
			"true|404",
		},
		{
			"fetched data can define records",
			`FETCH("URL/d")
				.then(function (r) { return r.json(); })
				.then(function (j) {
					D("x.com", "none", TXT("@", "first"), TXT("m", j.method));
				});`,
			"first",
		},
	}
	for _, tst := range tests {
		t.Run(tst.desc, func(t *testing.T) {
			script := strings.ReplaceAll(tst.script, "URL", srv.URL)
			testifyrequire.Equal(t, tst.want, runTXT(t, script))
		})
	}
}

func TestFetchDisabled(t *testing.T) {
	old := EnableFetch
	EnableFetch = false
	t.Cleanup(func() { EnableFetch = old })

	for _, fn := range []string{"fetch", "FETCH"} {
		t.Run(fn, func(t *testing.T) {
			_, err := ExecuteJavascriptString([]byte(fn+`("http://127.0.0.1:1/")`), true, nil)
			testifyrequire.ErrorContains(t, err, "fetch")
			testifyrequire.ErrorContains(t, err, "not defined")
		})
	}
}

// TestFetchFailurePanics checks that a failed FETCH() exits the process
// with status 1 (FETCH catches the rejection with PANIC). The script runs
// in a child process because PANIC calls os.Exit.
func TestFetchFailurePanics(t *testing.T) {
	if os.Getenv("DNSCONTROL_TEST_FETCH_CHILD") == "1" {
		EnableFetch = true
		// Port 1 on localhost refuses the connection.
		_, err := ExecuteJavascriptString([]byte(`FETCH("http://127.0.0.1:1/")`), true, nil)
		fmt.Fprintf(os.Stderr, "FETCH did not exit: %v\n", err)
		os.Exit(0)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestFetchFailurePanics$")
	cmd.Dir = "pkg/js" // init() changes to "../.." again in the child.
	cmd.Env = append(os.Environ(), "DNSCONTROL_TEST_FETCH_CHILD=1")
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	testifyrequire.ErrorAs(t, err, &exitErr, "child output: %s", out)
	testifyrequire.Equal(t, 1, exitErr.ExitCode(), "child output: %s", out)
	testifyrequire.Contains(t, string(out), "127.0.0.1:1")
}

// TestUnderscore checks that underscore.js stays available to user
// scripts; documentation/advanced-features/code-tricks.md uses _.each.
func TestUnderscore(t *testing.T) {
	got := runTXT(t, `
		var seen = [];
		_.each(["a.com", "b.com"], function (d) { seen.push(d); });
		var doubled = _.map([1, 2, 3], function (x) { return x * 2; });
		var odd = _.filter([1, 2, 3], function (x) { return x % 2; });
		D("x.com", "none", TXT("@", [
			typeof _, seen.join(","), doubled.join(","), odd.join(","),
			_.isArray([]), _.isString("s"), _.keys({ k: 1 }).join(",")
		].join("|")));`)
	testifyrequire.Equal(t, "function|a.com,b.com|2,4,6|1,3|true|true|k", got)
}

// TestHelpersWithoutUnderscore checks that helpers.js does not depend on
// underscore.js, by removing _ before calling helpers that used it.
func TestHelpersWithoutUnderscore(t *testing.T) {
	conf, err := ExecuteJavascriptString([]byte(`
		_ = undefined;
		var TRANSFORM_BASE = [{ low: "0.0.0.0", high: "1.1.1.1", newBase: ["2.2.2.2", "3.3.3.3"] }];
		D("x.com", "none", { custom: "meta" }, DefaultTTL("1d"),
			A("a", "1.2.3.4", TTL("1h"), { rmeta: "yes" }),
			TXT("t", ["seg1", "seg2"]),
			R53_ALIAS("r", "A", "x.com.", R53_EVALUATE_TARGET_HEALTH(true)),
			IMPORT_TRANSFORM(TRANSFORM_BASE, "y.com", 60),
			DKIM_BUILDER({ selector: "s1", pubkey: "abc", hashtypes: "sha256", flags: ["y", "s"] }),
			DMARC_BUILDER({ policy: "reject" })
		);`), true, nil)
	testifyrequire.NoError(t, err)
	testifyrequire.Len(t, conf.Domains, 1)
	dc := conf.Domains[0]
	testifyrequire.Equal(t, "meta", dc.Metadata["custom"])

	got := map[string]string{}
	for _, r := range dc.Records {
		got[r.Name] = fmt.Sprintf("%s %d %s", r.Type, r.TTL, r.ComparableV3)
	}
	testifyrequire.Equal(t, "A 3600 1.2.3.4", got["a"])
	testifyrequire.Equal(t, `TXT 86400 "seg1seg2"`, got["t"])
	testifyrequire.Equal(t, `R53_ALIAS 86400 A x.com. true ""`, got["r"])
	testifyrequire.Equal(t, `TXT 86400 "v=DKIM1; h=sha256; p=abc; t=y:s"`, got["s1._domainkey"])
	testifyrequire.Equal(t, `TXT 86400 "v=DMARC1; p=reject"`, got["_dmarc"])

	// IMPORT_TRANSFORM's data is only visible in its JSON form.
	var transformJSON []byte
	for _, r := range dc.Records {
		if r.Type == "IMPORT_TRANSFORM" {
			transformJSON, err = json.Marshal(r)
			testifyrequire.NoError(t, err)
		}
	}
	testifyrequire.Contains(t, string(transformJSON), `0.0.0.0 ~ 1.1.1.1 ~ 2.2.2.2,3.3.3.3 ~ \" 60 \"\" y.com`)
}

func TestBuiltinArgumentErrors(t *testing.T) {
	tests := []struct{ desc, script, wantErr string }{
		{"require no args", `require()`, "require takes exactly one argument"},
		{"require two args", `require("a.js", "b.js")`, "require takes exactly one argument"},
		{"require missing file", `require("pkg/js/parse_tests/nonexistent.js")`, "no such file or directory"},
		{"glob no args", `glob()`, "glob requires at least one argument"},
		{"glob too many args", `glob("a", true, ".js", 4)`, "glob requires at least one argument"},
		{"glob non-string path", `glob(5)`, "glob: first argument needs to be a path"},
		{"glob empty path", `glob("")`, "glob: first argument needs to be a path"},
		{"glob non-bool recursive", `glob("pkg/js", "yes")`, "glob: second argument, if recursive, needs to be bool."},
		{"glob non-string extension", `glob("pkg/js", true, 5)`, "glob: third argument, file extension, needs to be a string."},
		{"glob missing dir", `glob("nonexistent-dir/")`, "glob: provided path does not exist."},
		{"REV no args", `REV()`, "REV takes exactly one argument"},
		{"REV bad input", `REV("notanip")`, "not an IP address"},
		{"REVCOMPAT no args", `REVCOMPAT()`, "REVCOMPAT takes exactly one argument"},
		{"REVCOMPAT bad mode", `REVCOMPAT("bogus")`, "bogus"},
		{"HASH one arg", `HASH("sha1")`, "HASH takes exactly two arguments"},
		{"HASH bad algorithm", `HASH("md5", "x")`, "invalid algorithm md5 given"},
		{"PANIC no args", `PANIC()`, "PANIC takes exactly one argument"},
		{"syntax error", `D("x.com", "none"`, ""},
		{"throw string", `throw "custom failure"`, "custom failure"},
		{"throw Error", `throw new Error("custom error")`, "custom error"},
	}
	for _, tst := range tests {
		t.Run(tst.desc, func(t *testing.T) {
			resetDirectories(t)
			_, err := ExecuteJavascriptString([]byte(tst.script), true, nil)
			testifyrequire.Error(t, err)
			testifyrequire.ErrorContains(t, err, tst.wantErr)
		})
	}
}

func TestBuiltinResults(t *testing.T) {
	tests := []struct{ desc, script, want string }{
		{"REV IPv4", `REV("1.2.3.0/24")`, "3.2.1.in-addr.arpa"},
		{"REV IPv6", `REV("2001:db8::/32")`, "8.b.d.0.1.0.0.2.ip6.arpa"},
		{"HASH sha1", `HASH("sha1", "abc")`, "a9993e364706816aba3e25717850c26c9cd0d89d"},
		{"HASH SHA256", `HASH("SHA256", "abc")`, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"glob non-recursive", `glob("pkg/js/parse_tests/complexImports", false).join(",")`,
			"pkg/js/parse_tests/complexImports/base.js"},
		{"glob recursive by default", `glob("pkg/js/parse_tests/complexImports/a").join(",")`,
			"pkg/js/parse_tests/complexImports/a/a.js,pkg/js/parse_tests/complexImports/a/c/c.js"},
		{"glob extension without dot", `glob("pkg/js/parse_tests/complexImports/a", true, "js").length`, "2"},
		{"require of .js returns true", `require("pkg/js/parse_tests/complexImports/a/a.js")`, "true"},
		{"nested relative require", `require("pkg/js/parse_tests/complexImports/a/c/c.js") && typeof a + typeof c`, "functionfunction"},
		{"require of .json returns the value", `typeof require("pkg/js/parse_tests/domain-ip-map.json")`, "object"},
		{"require of .json5 returns the value", `typeof require("pkg/js/parse_tests/domain-ip-map.json5")`, "object"},
	}
	for _, tst := range tests {
		t.Run(tst.desc, func(t *testing.T) {
			testifyrequire.Equal(t, tst.want, runTXT(t, `D("x.com", "none", TXT("@", ""+(`+tst.script+`)));`))
		})
	}
}

func TestCLIVariables(t *testing.T) {
	conf, err := ExecuteJavascriptString([]byte(`D("x.com", "none", TXT("@", myvar + "|" + typeof myvar));`),
		true, map[string]string{"myvar": "hello"})
	testifyrequire.NoError(t, err)
	testifyrequire.Equal(t, "hello|string", conf.Domains[0].Records[0].GetTargetTXTJoined())
}

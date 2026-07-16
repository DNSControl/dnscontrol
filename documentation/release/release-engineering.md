# How to build and ship a release

These are the instructions for producing a release.

GitHub Actions (GHA) will do most of the work for you. You will need to edit the draft release notes and click a button to make the release public.

Please change the version number as appropriate.  Substitute (for example) `v4.2.0` any place you see `$VERSION` in this doc.

## Step 0. Update dependencies

```shell
git checkout main
git checkout -b update_deps
go install github.com/oligot/go-mod-upgrade@latest
go-mod-upgrade
go mod tidy
git commit -m "CHORE: Update dependencies" go.sum go.mod
```

## Step 1. Rebuild generated files

```shell
git fetch origin main
git reset --hard origin/main
git checkout -b generate
bin/generate-all.sh
git status
git commit -am "CHORE: generate-all.sh"
```

## Step 2. Tag the commit in main that you want to release

```shell
git checkout main
git pull --rebase --ff-only --prune
git tag -l |grep -F v4. | sort --version-sort --field-separator=. --key=2,2 | tail
export VERSION=v4.x.0
git tag -m "Release $VERSION" -a $VERSION
git push origin HEAD --tags
```

Soon after GitHub will start an [Action](https://github.com/DNSControl/dnscontrol/actions) Workflow called "draft release" which will build all release binaries and write the draft release notes.

## Step 3. Create the release notes

The draft release notes are created for you. In this step you'll edit them.

The GHA workflow uses [GoReleaser](https://goreleaser.com/) which produces the [GitHub Release](https://github.com/DNSControl/dnscontrol/releases) with Release Notes derived from the commit history between now and the last tag. These notes are just a draft and needs considerable editing. These release notes are used elsewhere, in particular the email step.

Release notes style guide:

* Entries in the bullet list should be phrased in the positive: "Feature FOO now does BAR".  This is often the opposite of the related issue, which was probably phrased, "Feature FOO is broken because of BAR".
* Every item should include the ID of the issue related to the change. If there was no issue, create one and close it.
* Sort the list most important/exciting changes earlier in the list.
* Items related to a specific provider should begin with the all-caps name of the provider, such as "ROUTE53: Added support for sandwiches (#100)"
* The `Deprecation warnings` section should just copy from `README.md`.  If you change one, change it in the README too (you can make that change in this PR).

See [https://github.com/DNSControl/dnscontrol/releases](https://github.com/DNSControl/dnscontrol/releases) for examples for recent release notes and copy that style.

## Step 4. Announce it via email

Email the release notes to the mailing list: (note the format of the Subject line and that the first line of the email is the URL of the release)

```text
To: dnscontrol-discuss@googlegroups.com
Subject: New release: dnscontrol v$VERSION

https://github.com/DNSControl/dnscontrol/releases/tag/v$VERSION

[insert the release notes here]
```

{% hint style="info" %}
**NOTE**: You won't be able to post to the mailing list unless you are on
it.  [Click here to join](https://groups.google.com/g/dnscontrol-discuss).
{% endhint %}

## Tip: How to bump the major version

If you bump the major version, you need to change all the source files.  The last time this was done (v3 -> v4) these two commands were used. They're included her for reference.

```shell
#  Make all the changes:
sed -i.bak -e 's@github.com/DNSControl/dnscontrol.v3@github.com/DNSControl/dnscontrol/v4@g' go.* $(fgrep -lri --include '*.go' github.com/DNSControl/dnscontrol/v3 *)
# Delete the backup files:
find * -name \*.bak -delete
```

## Tip: Configuring GHA integration tests

### Overview

GHA is configured to run an integration test for any provider listed in the "provider" list. However the test is skipped if the `*_DOMAIN` variable is not set. For example, the Google Cloud provider integration test is only run if `GCLOUD_DOMAIN` is set.

* Q: What labels control the integration tests?
* A: A PR only runs a "smoke test" (the first few tests).  Add the label "fulltest" to run all tests. (The daily run of integration tests on the main branch always does all test.)

* Q: Where are non-secret environment variables stored?
* A: GHA calls them "Variables". Update them here: https://github.com/DNSControl/dnscontrol/settings/variables/actions

* Q: Where are SECRET environment variables stored?
* A: GHA calls them "Secrets". Update them here: https://github.com/DNSControl/dnscontrol/settings/secrets/actions

### How do I add a single new integration test?

1. Ensure the provider has an entry in `integrationTest/profiles.json`.
2. Set the `FOO_DOMAIN` variables in GHA via https://github.com/DNSControl/dnscontrol/settings/variables/actions
3. All other variables should be stored as secrets (for consistency).  Add them to the `integration-tests` section of `.github/workflows/pr_integration_tests.yml`. Set them in GHA via https://github.com/DNSControl/dnscontrol/settings/secrets/actions

### How do I add a "bring your own keys" integration test?

Overview: You will fork the repo and add any secrets to your fork.  For security reasons you won't have access to the secrets from the main repository.

1. [Fork DNSControl/dnscontrol](https://github.com/DNSControl/dnscontrol/fork) in GitHub.

    If you already have a fork, be sure to use the "sync fork" button on the main page to sync with the upstream.

2. In your fork, set the `${DOMAIN}_DOMAIN` variable in GHA via Settings :: Secrets and variables :: Actions :: Variables.

3. In your fork, set any secrets in GHA via Settings :: Secrets and variables :: Actions :: Secrets.

5. Start a build

## Tip: How to rebuild flattener

Rebuilding flatter requires go1.17.1 and the gopherjs compiler.

Install go1.17.1:

```shell
go install golang.org/dl/go1.17.1@latest
go1.17.1 download
```

Install [GopherJS](https://github.com/gopherjs/gopherjs):

```shell
go install github.com/gopherjs/gopherjs@latest
```

Build the software:

{% hint style="info" %}
**NOTE**: GOOS can't be Darwin because GOPHERJS doesn't support it.
{% endhint %}

```shell
cd docs/flattener/
export GOPHERJS_GOROOT="$(go1.17.1 env GOROOT)"
export GOOS=linux
gopherjs build
```

## Tip: How to update modules

List out-of-date modules and update any that seem worth updating:

```shell
go install github.com/oligot/go-mod-upgrade@latest
go-mod-upgrade
go mod tidy
```

OLD WAY:

```shell
go install github.com/psampaz/go-mod-outdated@latest
go list -mod=mod -u -m -json all | go-mod-outdated -update -direct

# If any are out of date, update via:

go get module/path

# Once the updates are complete, tidy up:

go mod tidy
```

## Tip: How to test GoReleaser

(These are random notes)

```
git tag -a v4.42.3-rc1 -m "Release candidate 4.42.3-rc1"
or
git tag -a v4.42.3 -m "Release candidate 4.42.3"
```

DO NOT PUSH THIS TAG. It should stay local. If you push it, GHA will build a release!

When done, delete the tag with:

```
git tag -d v4.42.3-rc1
or
git tag -d v4.42.3
```

```
touch /tmp/empty-notes.md
unset GITHUB_TOKEN
goreleaser release --clean --skip=publish,validate,announce --release-notes=/tmp/empty-notes.md --verbose
ls dist*
```

```
GITHUB_TOKEN=dummy goreleaser release --clean --skip=publish,announce --verbose 2>&1 | tee /tmp/goreleaser.log
```

Review output for homebrew/docker logs:

```
grep -i -A2 "homebrew\|cask" /tmp/goreleaser.log
grep -i -A2 "docker" /tmp/goreleaser.log
```

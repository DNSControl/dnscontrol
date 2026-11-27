# How to build and ship a release

- [Step 1: Verify everything is up to date](#step-1-verify-everything-is-up-to-date)
  - [Automated](#automated)
  - [Manual](#manual)
- [Step 2: Cut the release](#step-2-cut-the-release)
  - [Create an empty PR for the release](#create-an-empty-pr-for-the-release)
  - [Start the release automation](#start-the-release-automation)
- [Manual (escape hatch)](#manual-escape-hatch)
- [Release it to the public](#release-it-to-the-public)
- [What the release workflow guarantees, and how to verify it](#what-the-release-workflow-guarantees-and-how-to-verify-it)
- [Step 3: Create the release notes](#step-3-create-the-release-notes)
- [Step 4: Announce it via email](#step-4-announce-it-via-email)
- [Tip: How to bump the major version](#tip-how-to-bump-the-major-version)
- [Tip: Configuring GHA integration tests](#tip-configuring-gha-integration-tests)
  - [Overview](#overview)
  - [How do I add a single new integration test?](#how-do-i-add-a-single-new-integration-test)
  - [How do I add a "bring your own keys" integration test?](#how-do-i-add-a-bring-your-own-keys-integration-test)
- [Tip: How to rebuild flattener](#tip-how-to-rebuild-flattener)
- [Tip: How to update modules](#tip-how-to-update-modules)
- [Tip: How to test GoReleaser](#tip-how-to-test-goreleaser)

These are the instructions for producing a release.

GitHub Actions (GHA) will do most of the work for you. You will need to edit the draft release notes and click a button to make the release public.

Please change the version number as appropriate.  Substitute (for example) `v4.2.0` any place you see `$VERSION` in this doc.

## Step 1: Verify everything is up to date

### Automated

This script will run `bin/generate-all.sh` and prompt to upgrade depenencies.
It must be run from branch `main` or `prep_release`.

```shell
git checkout main
git config remote.origin.prune true ; git config fetch.prune true
git pull --rebase --ff-only --prune
bin/prepare_release.sh
```

### Manual

Dependencies:

```shell
git checkout main
git checkout -b update_deps
go install github.com/oligot/go-mod-upgrade@latest
go-mod-upgrade
go mod tidy
git commit -m "CHORE: update dependencies" go.sum go.mod
```

Generated files, linting, etc:

```shell
git fetch origin main
git reset --hard origin/main
git checkout -b generate
bin/generate-all.sh
git status
git commit -am "CHORE: generate-all.sh"
```

## Step 2: Cut the release

Pick the next release number:

```shell
git tag -l |grep -F v5. | sort --version-sort --field-separator=. --key=2,2 | tail
```

The manual dance below (empty PR → wait for tests → merge → tag) is now done by
creating an empty "release" PR, then a single GitHub Actions run.

### Create an empty PR for the release

```shell
git fetch origin main
git checkout main
git config remote.origin.prune true ; git config fetch.prune true
git pull --rebase --ff-only --prune
git reset --hard origin/main
git checkout -b "release_$VERSION"
git commit --allow-empty -m "Release $VERSION"
git push
gh pr create --base main --title "Release $VERSION" --body ""
```

### Start the release automation

1. Go to **Actions → "RELEASE: Make release candidate" → Run workflow**.
2. In the **"Use workflow from"** dropdown, choose the branch to release from (usually `main`; any branch is supported).
3. Fill in the inputs:
   - **version** (required): e.g. `v4.44.0` or `v4.44.0-rc1`. The run **fails fast if that tag already exists**.
   - **previous_tag** (optional): the tag the changelog compares against. Leave blank to auto-detect the most recent non-RC release (almost always what you want; set it only when releasing from an unusual branch).
   - **skip_longtest** (optional, default off): **danger** — skips the full `longtest` integration gate. Use only for an emergency release or when the suite is known-broken. When set, the run is loudly annotated and the job summary records that tests were skipped. (`preflight` still runs, so an invalid version or an existing tag still blocks the release.)
4. Click **Run workflow**.

The workflow then, in order:

1. **preflight** — validates the version; refuses if the tag already exists.
2. **verify** — runs the **entire** `longtest` integration suite as a gate. If anything fails, nothing is tagged or published.
3. **release** — tags the tip of the selected branch and runs GoReleaser to produce the **draft** release.

The release is tagged directly (not via an empty `Release <version>` commit):
this org forbids GitHub Actions from opening pull requests and `main` is
protected, so a PAT-free automated PR is not possible. The changelog range is
still correct because it is computed by version, not by commit ancestry.

## Manual (escape hatch)

You can still do the release by hand. Pushing a tag runs GoReleaser directly, but it
**skips** the integration-test gate — so run/verify tests yourself first.

```shell
# Set the version we are going to release

export VERSION=v4.43.2

# Create an empty PR for the release

git fetch origin main
git checkout main
git config remote.origin.prune true ; git config fetch.prune true
git pull --rebase --ff-only --prune
git reset --hard origin/main
git checkout -b "release_$VERSION"
git commit --allow-empty -m "Release $VERSION"
git push
gh pr create --base main --title "Release $VERSION" --body ""
```

Wait to tests to complete and merge.

```shell
gh run list -b "release_$VERSION"
gh run watch
```

WAIT for the GHA to complete. If there are errors, stop and fix them.

Merge it either manually or with this command:

```shell
gh pr merge --squash --delete-branch  $PR
```

Create the release

```shell
git fetch origin main
git checkout main
git config remote.origin.prune true ; git config fetch.prune true
git pull --rebase --ff-only --prune
git tag -m "Release $VERSION" -a $VERSION
git push origin HEAD --tags
```

Soon after GitHub will start an [Action](https://github.com/DNSControl/dnscontrol/actions) Workflow called "draft release" which will build all release binaries and write the draft release notes.

Wait to tests to complete and merge.

```shell
gh run list -b "$VERSION"
gh run watch
```

WAIT for the GHA to complete. If there are errors, stop and fix them.

## Release it to the public

Find the release and edit the notes.

```text
https://github.com/DNSControl/dnscontrol/releases
```

When you submit it:

- "Pre-Release" for rc releases, "Latest" for real releases.
- Create a discussion for this release

## What the release workflow guarantees, and how to verify it

Every release carries three layers of cryptographic supply-chain evidence.
All three are keyless: the signing identity is the release workflow itself
(`DNSControl/dnscontrol/.github/workflows/release_draft.yml`), certified by
GitHub's OIDC issuer and recorded in the public Sigstore transparency log.
There are no signing keys to manage or trust.

- **cosign signature** — `checksums.txt.sigstore.json`, a keyless Sigstore
  bundle over `checksums.txt`, uploaded to the release page. Consumed by
  `cosign verify-blob`.
- **SLSA provenance** — `multiple.intoto.jsonl`, a SLSA Level 3 statement
  attached to the release page by the `upload-provenance` job. Consumed by
  `slsa-verifier verify-artifact`; this is also the asset OpenSSF
  Scorecard's Signed-Releases check looks for (issue #3468).
- **GitHub attestation** — build provenance for every release artifact,
  registered with GitHub's attestation API by the
  `Generate build attestations` step. Consumed by
  `gh attestation verify`.

### Why both SLSA provenance and GitHub attestations?

The two mechanisms are intentional and not redundant — their consumers
differ:

- `gh attestation verify` talks only to GitHub's attestation API. Before
  the `Generate build attestations` step existed, it returned HTTP 404 for
  every dnscontrol release.
- OpenSSF Scorecard's Signed-Releases check (issue #3468) looks for a
  signature or provenance file on the release page, not in the
  attestation API.
- `slsa-verifier` consumes the `multiple.intoto.jsonl` asset; it does not
  use the attestation API.

Don't "unify" the two mechanisms.

### Post-release verification checklist

After a release ships, walk all layers (the same commands the release
notes' "Verify downloads" footer documents):

```shell
VERSION=5.10.1   # example: the version just released
FILE=dnscontrol_${VERSION}_linux_amd64.tar.gz
gh release download "v${VERSION}" --repo DNSControl/dnscontrol \
  --pattern "$FILE" --pattern checksums.txt \
  --pattern 'checksums.txt.sigstore.json' --pattern 'multiple.intoto.jsonl'

# 1. cosign bundle over checksums.txt:
cosign verify-blob checksums.txt \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp 'https://github\.com/DNSControl/dnscontrol/\.github/workflows/release_draft\.yml@refs/(tags|heads)/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

# 2. GitHub attestation for any release asset (including checksums.txt):
gh attestation verify "$FILE" --repo DNSControl/dnscontrol

# 3. SLSA provenance. Releases cut via the release workflow
#    (workflow_dispatch) record the BRANCH they were cut from, so use
#    --source-branch. Releases cut by manually pushing a tag record the
#    tag instead; use --source-tag for those.
slsa-verifier verify-artifact \
  --provenance-path multiple.intoto.jsonl \
  --source-uri git+https://github.com/DNSControl/dnscontrol \
  --source-branch main \
  "$FILE"

# 4. checksums:
sha256sum -c checksums.txt --ignore-missing
```

Notes:

- `multiple.intoto.jsonl` is a Sigstore bundle (signature, certificate,
  and DSSE envelope in one file) — the stock output of
  `slsa-github-generator` v2.x. It is intentionally shipped as-is, not
  rewritten to a bare DSSE envelope: `slsa-verifier` 2.7.0+ reads it
  directly. Older `slsa-verifier` versions cannot parse the bundle;
  unwrap the DSSE envelope first
  (`jq -c '.dsseEnvelope' multiple.intoto.jsonl > multiple.dsse.jsonl`)
  and pass that file to `--provenance-path`.
- The `refs/(tags|heads)/` pattern in the cosign check matches both
  release entry paths: a manually pushed tag signs with a `refs/tags/`
  identity, a dispatch run with a `refs/heads/` one.
- Releases cut before the `Generate build attestations` step landed have
  no attestations registered; verify those via the cosign bundle and
  `checksums.txt`, as their release notes describe.

### History

- #3468 — OpenSSF Scorecard flagged releases as unsigned; the original
  request for release signing.
- #3563 (closed, May 2025) — first attempt: SLSA provenance via the
  generator's built-in uploader. The uploader created a second draft
  release on repos with many releases, and the approach was abandoned.
- #5011 (merged, Oct 2026) — cosign keyless signing plus SLSA
  provenance. The generator's uploader is replaced by the
  `upload-provenance` job, which attaches `multiple.intoto.jsonl` with
  `gh release upload` (resolves the draft by pending tag name and is
  idempotent).
- This change — the `Generate build attestations` step, so
  `gh attestation verify` works on every release, plus these
  verification docs.

## Step 3: Create the release notes

The draft release notes are created for you. In this step you'll edit them.

The GHA workflow uses [GoReleaser](https://goreleaser.com/) which produces the [GitHub Release](https://github.com/DNSControl/dnscontrol/releases) with Release Notes derived from the commit history between now and the last tag. These notes are just a draft and needs considerable editing. These release notes are used elsewhere, in particular the email step.

Release notes style guide:

- Entries in the bullet list should be phrased in the positive: "Feature FOO now does BAR".  This is often the opposite of the related issue, which was probably phrased, "Feature FOO is broken because of BAR".
- Every item should include the ID of the issue related to the change. If there was no issue, create one and close it.
- Sort the list most important/exciting changes earlier in the list.
- Items related to a specific provider should begin with the all-caps name of the provider, such as "ROUTE53: Added support for sandwiches (#100)"
- The `Deprecation warnings` section should just copy from `README.md`.  If you change one, change it in the README too (you can make that change in this PR).

See [https://github.com/DNSControl/dnscontrol/releases](https://github.com/DNSControl/dnscontrol/releases) for examples for recent release notes and copy that style.

## Step 4: Announce it via email

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

- Q: What labels control the integration tests?
- A: A PR only runs a "smoke test" (the first few tests).  Add the label "fulltest" to run all tests. (The daily run of integration tests on the main branch always does all test.)

- Q: Where are non-secret environment variables stored?
- A: GHA calls them "Variables". Update them here: https://github.com/DNSControl/dnscontrol/settings/variables/actions

- Q: Where are SECRET environment variables stored?
- A: GHA calls them "Secrets". Update them here: https://github.com/DNSControl/dnscontrol/settings/secrets/actions

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

4. Start a build

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

```shell
git tag -a v4.42.3-rc1 -m "Release candidate 4.42.3-rc1"
# or
git tag -a v4.42.3 -m "Release candidate 4.42.3"
```

DO NOT PUSH THIS TAG. It should stay local. If you push it, GHA will build a release!

When done, delete the tag with:

```shell
git tag -d v4.42.3-rc1
or
git tag -d v4.42.3
```

```shell
touch /tmp/empty-notes.md
unset GITHUB_TOKEN
goreleaser release --clean --skip=publish,validate,announce --release-notes=/tmp/empty-notes.md --verbose
ls dist*
```

```shell
GITHUB_TOKEN=dummy goreleaser release --clean --skip=publish,announce --verbose 2>&1 | tee /tmp/goreleaser.log
```

Review output for homebrew/docker logs:

```shell
grep -i -A2 "homebrew\|cask" /tmp/goreleaser.log
grep -i -A2 "docker" /tmp/goreleaser.log
```

## Tip: How to update codeberg.org/miekg/dns

Currently we're using a patched version of codeberg.org/miekg/dns because we're
waiting for https://codeberg.org/miekg/dns/pulls/1074 to land.

Any time that PR changes, we need to update `go.mod`

```
go mod edit -dropreplace  codeberg.org/miekg/dns
GONOSUMDB=codeberg.org/TomOnTime/dns go mod edit -replace codeberg.org/miekg/dns=codeberg.org/TomOnTime/dns@tlim_txtfull
GONOSUMDB=codeberg.org/TomOnTime/dns GOPROXY=direct go mod tidy
```


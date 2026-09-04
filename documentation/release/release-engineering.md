# How to build and ship a release

DNSControl releases are **automated** (see [issue #4829](https://github.com/DNSControl/dnscontrol/issues/4829)). Day-to-day development happens on the `develop` branch. **A release is cut by merging `develop` into `main`** — that merge is the only human action required. Everything else (version number, tag, changelog, binaries, packages, Docker images, Homebrew tap, and the published GitHub Release) happens automatically.

- [How to build and ship a release](#how-to-build-and-ship-a-release)
  - [The model in one picture](#the-model-in-one-picture)
  - [Contributor rules: Conventional Commits](#contributor-rules-conventional-commits)
  - [How to cut a release](#how-to-cut-a-release)
  - [Emergency: skip the integration gate](#emergency-skip-the-integration-gate)
  - [What happens automatically](#what-happens-automatically)
  - [Manual escape hatch](#manual-escape-hatch)
  - [One-time setup / repo settings](#one-time-setup--repo-settings)
  - [Tip: How to bump the major version](#tip-how-to-bump-the-major-version)
  - [Tip: Configuring GHA integration tests](#tip-configuring-gha-integration-tests)
    - [Overview](#overview)
    - [How do I add a single new integration test?](#how-do-i-add-a-single-new-integration-test)
    - [How do I add a "bring your own keys" integration test?](#how-do-i-add-a-bring-your-own-keys-integration-test)
  - [Tip: How to rebuild flattener](#tip-how-to-rebuild-flattener)
  - [Tip: How to update modules](#tip-how-to-update-modules)
  - [Tip: How to test GoReleaser](#tip-how-to-test-goreleaser)

## The model in one picture

```text
   feature PRs  ─(squash, Conventional Commit title)─▶  develop
                                                          │
                                    you open & merge PR   │  (merge commit, NOT squash)
                                    (full longtest gate)  ▼
                                                         main
                                                          │  push to main
                                    svu computes version  │  → tag vX.Y.Z
                                    GoReleaser publishes   ▼
                                                     GitHub Release
```

- **`develop`** — the integration branch. All contributor PRs squash-merge here.
- **`main`** — the release branch. It only ever receives the `develop -> main` merge.

## Contributor rules: Conventional Commits

PR **titles** must follow [Conventional Commits](https://www.conventionalcommits.org/). PRs are squash-merged into `develop` and GitHub uses the PR title as the commit subject, so the title is what drives the version bump and the changelog. This is enforced by the `PR: Conventional title` check on every PR into `develop`.

Format: `type(scope): description`

- **Types** that trigger a release: `feat` (→ minor), `fix` (→ patch). A trailing `!` (e.g. `feat!:`) or a `BREAKING CHANGE` footer → major.
- **Types** that do **not** trigger a release: `docs`, `chore`, `ci`, `build`, `refactor`, `perf`, `test`, `revert`.
- **Provider-specific changes** use the scope **`p/<name>`**, e.g.:

  ```text
  fix(p/route53): stop sending unchanged CAA records
  feat(p/azure_dns): support private zones
  ```

  The `p/` scope is the *only* thing that marks a change as provider-specific. There is **no per-provider list to maintain** anywhere — adding a new provider needs no changelog or config edits. Non-provider scopes are free-form (`feat(spf):`, `fix(core):`, `chore(deps):`, …).

## How to cut a release

1. Open a pull request from `develop` into `main`.
2. Wait for the **`release-gate`** check to go green. It runs the **entire** `longtest` integration suite (this is why it can be slow — and why releases are batched rather than one-per-merge).
3. **Merge it as a merge commit — NOT a squash.** Squashing would collapse every `feat`/`fix` into one commit and destroy both the version computation and the changelog.

That's it. The merge lands on `main`, which triggers the automation below.

## Emergency: skip the integration gate

If you must release without waiting for the full `longtest` suite (e.g. the suite is broken by an external provider outage, or a hotfix must ship now):

- Add the label **`skip-longtest`** to the `develop -> main` PR.
- The `release-gate` check then reports success **without** running the suite, but records a loud warning and a job-summary note so the bypass is auditable in the PR.
- Merge as usual.

The fully-manual escape hatch below is also always available.

## What happens automatically

On every push to `main` (i.e. every `develop -> main` merge), `.github/workflows/release_on_merge.yml`:

1. Runs [`svu`](https://github.com/caarlos0/svu) to compute the next version from the Conventional Commits since the last tag. If there is nothing releasable (only `chore`/`ci`/`docs`/etc.), it **stops** — no tag, no release.
2. Creates and pushes the annotated tag `vX.Y.Z`.
3. Runs [GoReleaser](https://goreleaser.com/), which builds every artifact and **publishes** the GitHub Release (no draft — `release.draft: false` in `.goreleaser.yml`).

The release notes are generated by GoReleaser from the commit history, grouped by Conventional-Commit type (Breaking / New features / Provider-specific / Bug fixes / Documentation / CI/CD / Dependencies / …). The standing "welcome", monthly-call announcement, deprecation warnings, and install instructions live in the GoReleaser `header`/`footer` in `.goreleaser.yml` — edit them there.

> **Tags are pushed with `GITHUB_TOKEN`**, which by design does *not* re-trigger workflows, so the `release_on_merge` job runs GoReleaser itself. That is why this does not double-fire with the tag-push escape hatch below.

## Manual escape hatch

You can still cut a release by hand — useful if `develop -> main` itself is wedged. Two ways, both in `.github/workflows/release_draft.yml`:

1. **Actions → "RELEASE: Make release candidate" → Run workflow.** Pick the branch (usually `main`), enter the `version` (e.g. `v5.1.0` or `v5.1.0-rc1`), optionally a `previous_tag`, and optionally `skip_longtest` (danger: skips the integration gate). This runs preflight → the full `longtest` gate → tag → GoReleaser.
2. **Push a tag** (`git push origin vX.Y.Z`). This runs GoReleaser directly and **skips** the integration gate, so verify tests yourself first.

Both paths use the same GoReleaser config, so they also publish (non-draft).

## One-time setup / repo settings

These are GitHub settings (not in the repo) that the model depends on:

- **Squash merges use the PR title** as the commit subject (Settings → General → Pull Requests → "Default commit message" → *Pull request title*).
- **`develop` and `main` both allow the right merge methods:** feature PRs into `develop` use *Squash*; the `develop -> main` PR uses *Create a merge commit*. Both methods must be enabled.
- **Branch protection on `main`** requires the `release-gate` status check.
- **Branch protection on `develop`** requires the `PR: Conventional title` check.
- The `skip-longtest` label must exist in the repo.

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

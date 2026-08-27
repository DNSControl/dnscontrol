# GitHub Actions

- [GitHub Actions](#github-actions)
  - [PR Checks Overview](#pr-checks-overview)
  - [Check: git status](#check-git-status)
    - [Check: go fmt](#check-go-fmt)
    - [Check: prettier](#check-prettier)
    - [Check: fmtjson](#check-fmtjson)
    - [Check: go mod tidy](#check-go-mod-tidy)
    - [Check: go generate](#check-go-generate)
    - [Check: go fix](#check-go-fix)
  - [Lint](#lint)
  - [Build \& Test](#build--test)
  - [Running all checks at once](#running-all-checks-at-once)

## PR Checks Overview

Every pull request runs the following GitHub Actions workflows. All checks must pass before a PR can be merged.

| Workflow | File | Description |
| --- | --- | --- |
| **Check git status** | `pr_check_git_status.yml` | Ensures all generated/formatted files are committed |
| **Lint** | `pr_lint.yml` | Runs `golangci-lint` |
| **Build & Test** | `pr_build.yml` | Runs unit tests and builds binaries via GoReleaser |

## Check: git status

The git status workflow runs formatting and code generation commands, then verifies no files were modified. If any command produces uncommitted changes, that specific check fails.

Each check runs as a **separate job**, so you can immediately see which one failed in the PR checks UI.

### Check: go fmt

**What it does:** Formats all Go source files using `go fmt`.

**How to fix locally:**

```bash
go fmt ./...
```

Commit the resulting changes.

### Check: prettier

**What it does:** Formats `pkg/js/helpers.js` using [Prettier](https://prettier.io/).

**How to fix locally:**

```bash
npm install
node_modules/.bin/prettier --write pkg/js/helpers.js
```

Commit the resulting changes. Prettier configuration is in `.prettierrc`.

### Check: fmtjson

**What it does:** Formats all JSON files in the repository (except `package-lock.json`) using `bin/fmtjson`.

**How to fix locally:**

```bash
bin/fmtjson $(find . -path ./.vscode -prune -o -type f -name "*.json" ! -name "package-lock.json" -print)
```

Commit the resulting changes.

### Check: go mod tidy

**What it does:** Cleans up `go.mod` and `go.sum` by removing unused dependencies and adding missing ones.

**How to fix locally:**

```bash
go mod tidy
```

Commit the resulting changes to `go.mod` and `go.sum`.

### Check: go generate

**What it does:** Runs all `//go:generate` directives in the codebase. This generates TypeScript type definitions, the feature matrix, and the OWNERS file. Requires `stringer` to be installed.

**How to fix locally:**

```bash
go install golang.org/x/tools/cmd/stringer@latest
go generate ./...
```

Commit the resulting changes.

### Check: go fix

**What it does:** Runs `go fix` to update packages to use newer APIs when Go introduces changes.

**How to fix locally:**

```bash
go fix ./...
```

Commit the resulting changes.

## Lint

Runs [`golangci-lint`](https://golangci-lint.run/) with the configuration in `.golangci.yml`.

**How to fix locally:**

```bash
golangci-lint run ./...
```

See `.golangci.yml` for the list of enabled linters and their settings.

## Build & Test

Runs all unit tests with `gotestsum` and builds binaries for all platforms using GoReleaser.

**How to run locally:**

```bash
go test ./...
go build .
```

## Running all checks at once

The `bin/generate-all.sh` script runs most of the above checks in sequence:

```bash
bin/generate-all.sh
```

This is useful before committing to catch all formatting and generation issues at once.

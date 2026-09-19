# Contributing to DNSControl

Thank you for your interest in contributing to DNSControl! This guide will help you get started.

## Prerequisites

- **Go 1.26+** (see `go.mod` for the exact version)
- **golangci-lint** (optional, used by CI and `bin/generate-all.sh`)
- **staticcheck** (optional, used by `bin/generate-all.sh`)

## Building and testing

Build the binary:

```shell
go build .
```

Run all unit tests:

```shell
go test ./...
```

Run tests for a specific package:

```shell
go test ./pkg/spflib/
```

Run a single test:

```shell
go test ./pkg/spflib/ -run TestParseQualifiedMechanisms
```

Run the linter:

```shell
golangci-lint run
```

## Before committing

Run `bin/generate-all.sh` from the repository root. This script handles formatting, code generation, linting, and keeping generated files in sync:

```shell
bin/generate-all.sh
```

It runs `go fmt`, `go generate`, `go mod tidy`, JSON formatting, and optionally `golangci-lint` and `staticcheck` if they are installed.

## Pull request titles

DNSControl squash-merges pull requests, so the pull request title becomes the commit message on `main`. The title must follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/#summary): `type(scope): subject`. The `PR: Commitlint` check enforces this with the rules in `commitlint.config.js`. The commits inside your pull request are not checked.

| Type | Use for |
| --- | --- |
| `feat` | New functionality |
| `fix` | Bug fixes |
| `docs` | Documentation changes |
| `perf` | Performance improvements |
| `refactor` | Code refactoring |
| `style` | Code style changes |
| `test` | Test additions or changes |
| `build` or `ci` | Build and CI/CD changes |
| `chore` | Maintenance, dependency updates |

Rules:

- Provider-specific changes use the scope `p/PROVIDERNAME`, for example `fix(p/CLOUDFLAREAPI): correct TTL rounding` or `feat(p/ROUTE53): support alias records for NS`.
- Other scopes are optional and must be a single word, for example `chore(deps): update dependencies`.
- The subject starts with a lowercase letter. `fix(p/ROUTE53): Fix ...` fails the check.
- Mark a breaking change with `!` after the type or scope, for example `feat(p/BIND)!: ...`.

To check a title locally, run `npm install` once, then:

```shell
echo "fix(p/ROUTE53): correct TTL rounding" | npx commitlint
```

GoReleaser uses the type and scope to group the release changelog. `test` and `chore` changes are left out of the changelog. See `.goreleaser.yml` for the exact patterns.

## Integration tests

Integration tests run real DNS operations against a provider's API. They require credentials and a dedicated test zone. See the [integration test documentation](https://docs.dnscontrol.org/developer-info/integration-tests) for setup instructions.

```shell
go test ./integrationTest/ -v -args -provider PROVIDERNAME
```

## Writing a new provider

See [Writing new DNS providers](https://docs.dnscontrol.org/developer-info/writing-providers) for a step-by-step guide on implementing a new DNS provider.

Additional developer resources are available in the [developer info](https://docs.dnscontrol.org/developer-info/styleguide-code) section of the documentation.

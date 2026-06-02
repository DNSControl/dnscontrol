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

## Commit message conventions

Prefix your commit message title with one of the following categories:

| Prefix | Use for |
| --- | --- |
| `FEATURE:` | New functionality |
| `BUG:` | Bug fixes |
| `DOCS:` | Documentation changes |
| `CHORE:` or `MAINT:` | Maintenance, dependency updates |
| `BUILD:` or `CICD:` | CI/CD and build changes |
| `REFACTOR:` | Code refactoring |
| `TEST:` | Test additions or changes |
| `PROVIDERNAME:` | Provider-specific changes (e.g. `CLOUDFLAREAPI:`, `ROUTE53:`) |

These prefixes are used by GoReleaser to categorize the release changelog. See `.goreleaser.yml` for the full list of recognized patterns.

## Integration tests

Integration tests run real DNS operations against a provider's API. They require credentials and a dedicated test zone. See the [integration test documentation](https://docs.dnscontrol.org/developer-info/integration-tests) for setup instructions.

```shell
go test ./integrationTest/ -v -args -provider PROVIDERNAME
```

## Writing a new provider

See [Writing new DNS providers](https://docs.dnscontrol.org/developer-info/writing-providers) for a step-by-step guide on implementing a new DNS provider.

Additional developer resources are available in the [developer info](https://docs.dnscontrol.org/developer-info/styleguide-code) section of the documentation.

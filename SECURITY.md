# Security Policy

- [Security Policy](#security-policy)
  - [Supported Versions](#supported-versions)
  - [Reporting a Vulnerability](#reporting-a-vulnerability)
  - [Verifying Release Artifacts](#verifying-release-artifacts)

DNSControl is a command-line tool and therefore has a different (limited) attack surface as compared to a web app or other system.

## Supported Versions

Only the most recent release is supported with security updates.

When a major version is incremented, we'll support the previous major version for 6 months. For example, when v4.0 is released, we will support the most recent v3.x release for 6 months.

## Reporting a Vulnerability

To report a vulnerability please [create a new GitHub "issue"](https://github.com/DNSControl/dnscontrol/issues/new/choose).

We will respond in a best-effort manner, usually within 1 week. We will communicate via the GitHub issue unless we need to communicate privately, in which case we'll arrange a way to communicate directly.

## Verifying Release Artifacts

Every release is signed by the release workflow itself using
[Sigstore](https://sigstore.dev) keyless signatures (no signing keys to
manage), and carries [SLSA Level 3 provenance](https://slsa.dev/). The
quickest check, using the [GitHub CLI](https://cli.github.com), needs
nothing but your download:

```shell
gh attestation verify dnscontrol_X.Y.Z_linux_amd64.tar.gz \
  --repo DNSControl/dnscontrol
```

The full checklist — the cosign signature over `checksums.txt`, the
`multiple.intoto.jsonl` SLSA provenance via `slsa-verifier`, and the
sha256 comparison — is documented in the "Verify downloads" section of
every release (see the footer of the
[latest release](https://github.com/DNSControl/dnscontrol/releases/latest))
and in
[release-engineering.md](documentation/release/release-engineering.md).

Releases cut before the `Generate build attestations` step was added to
the release workflow have no attestations registered; verify those with
the cosign bundle and `checksums.txt`, as described in their own release
notes.

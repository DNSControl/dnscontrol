# GoReleaser

## Homebrew Tap

GoReleaser automatically publishes a Homebrew Cask to [DNSControl/homebrew-tap](https://github.com/DNSControl/homebrew-tap) on every release. This requires two components: a GitHub PAT for tap updates and macOS code signing + notarization.

### Homebrew TAP GitHub PAT

GoReleaser needs a GitHub Personal Access Token to push the Homebrew Cask formula to the `DNSControl/homebrew-tap` repository. This is a fine-grained PAT scoped to the `DNSControl` organization with minimal permissions.

| Item | Value |
|------|-------|
| **Secret name** | `HOMEBREW_TAP_TOKEN` (repository secret) |
| **Token type** | Fine-grained PAT |
| **Resource owner** | `DNSControl` (organization) |
| **Repository access** | `DNSControl/homebrew-tap` only |
| **Permissions** | Contents: Read and write |
| **Expires** | February 6, 2027 |
| **Action needed before** | ~January 18, 2027 |

**Links:**
- [GitHub Issue (tracking): Rotate Homebrew TAP GitHub PAT before Feb 6, 2027](https://github.com/DNSControl/dnscontrol/issues/4071)
- [Secret setting](https://github.com/DNSControl/dnscontrol/settings/secrets/actions/HOMEBREW_TAP_TOKEN)

#### Rotation procedure

1. Generate a new fine-grained PAT at [github.com/settings/personal-access-tokens/new](https://github.com/settings/personal-access-tokens/new):
   - **Resource owner**: `DNSControl`
   - **Repository access**: Only select repositories > `DNSControl/homebrew-tap`
   - **Permissions**: Contents > Read and write
   - **Expiration**: 1 year
2. Update the repository secret:
   ```bash
   gh secret set HOMEBREW_TAP_TOKEN \
     --repo DNSControl/dnscontrol \
     --body "<the-new-token>"
   ```
3. Verify that the next GoReleaser release successfully updates the Homebrew tap
4. Create a new tracking issue for the next rotation cycle

### macOS Code Signing & Notarization

Without code signing, macOS Gatekeeper shows an error on `brew install`:

> Apple could not verify "dnscontrol" is free of malware that may harm your Mac or compromise your privacy.

GoReleaser supports macOS notarization via the `notarize` section in `.goreleaser.yml`:

```yaml
notarize:
  macos:
    - enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'
      sign:
        certificate: "{{.Env.MACOS_SIGN_P12}}"
        password: "{{.Env.MACOS_SIGN_PASSWORD}}"
      notarize:
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"
        key_id: "{{.Env.MACOS_NOTARY_KEY_ID}}"
        key: "{{.Env.MACOS_NOTARY_KEY}}"
```

The `enabled` condition ensures that builds without secrets (e.g. local builds) continue normally.

#### Steps to activate

##### 1. Apple Developer Program

Sign up at [developer.apple.com/programs](https://developer.apple.com/programs/) ($99/year).

| Item | Value |
|------|-------|
| **Team Name** | JCID B.V. |
| **Team ID** | TY4QRVP7MM |
| **Expires** | February 10, 2027 |

##### 2. Developer ID Application Certificate

1. Open **Keychain Access** > **Certificate Assistant** > **Request a Certificate From a Certificate Authority...**
2. Choose **Saved to disk**, save the `.certSigningRequest` file
3. Go to [developer.apple.com/account/resources/certificates/add](https://developer.apple.com/account/resources/certificates/add)
4. Choose **Developer ID Application**, upload the `.certSigningRequest` file
5. Download the `.cer` file, double-click to import into Keychain

##### 3. Export as .p12

1. Open **Keychain Access**, find **Developer ID Application: [name]**
2. Right-click > **Export...** > format **.p12**
3. Set a strong password (this becomes `MACOS_SIGN_PASSWORD`)

##### 4. App Store Connect API Key

1. Go to [appstoreconnect.apple.com/access/integrations/api](https://appstoreconnect.apple.com/access/integrations/api)
2. **Generate API Key**, role: **Developer**
3. Download the `.p8` file (can only be downloaded once!)
4. Note the **Key ID** and **Issuer ID**

##### 5. GitHub Actions Secrets

Encode the `.p12` and `.p8` files:

```bash
base64 -i DeveloperIDApplication.p12 | pbcopy
base64 -i AuthKey_XXXXXX.p8 | pbcopy
```

Configure under repo > **Settings** > **Secrets and variables** > **Actions**:

| Secret | Value |
|---|---|
| `MACOS_SIGN_P12` | Base64-encoded `.p12` file |
| `MACOS_SIGN_PASSWORD` | Password of the `.p12` certificate |
| `MACOS_NOTARY_ISSUER_ID` | Issuer ID from App Store Connect |
| `MACOS_NOTARY_KEY_ID` | Key ID of the API key |
| `MACOS_NOTARY_KEY` | Base64-encoded `.p8` file |

##### 6. Testing

```bash
export MACOS_SIGN_P12=$(base64 -i DeveloperIDApplication.p12)
export MACOS_SIGN_PASSWORD="password"
export MACOS_NOTARY_ISSUER_ID="..."
export MACOS_NOTARY_KEY_ID="..."
export MACOS_NOTARY_KEY=$(base64 -i AuthKey_XXXXXX.p8)
goreleaser release --snapshot --clean
```

#### Background

- Homebrew `--no-quarantine` flag is deprecated since Homebrew 5.0.0 (November 2025)
- There is no cask-level option to disable quarantine
- Unsigned casks will be removed from the official Homebrew tap as of September 2026

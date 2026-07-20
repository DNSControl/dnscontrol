# How to add a requested provider

Follow the steps below to document a newly requested provider in the `documentation/provider/index.md` file.

## 1. Start from an up-to-date `main` branch

Make sure your local `main` branch is up to date, then create a new branch for your changes:

```shell
git switch main
git pull
git checkout -B docs/provider-request
```

## 2. Define variables

Set the following environment variables to use in the steps below:

```shell
export PROVIDER_NAME="Sav.com"
export GITHUB_ISSUE_NUMBER=3633
export GITHUB_FORK_REPO="yourusername/dnscontrol"
```

Replace `yourusername` with your actual GitHub username or organization name.

## 3. Edit the provider index file

Open the file in your preferred editor:

```shell
nano documentation/provider/index.md
```

Or, using PhpStorm:

```shell
phpstorm documentation/provider/index.md
```

Scroll to the **Requested providers** section and append the following line:

```markdown
* [Sav.com](https://github.com/DNSControl/dnscontrol/issues/3633) (#3633)
```

To generate this automatically, run:

```shell
echo "* [${PROVIDER_NAME}](https://github.com/DNSControl/dnscontrol/issues/${GITHUB_ISSUE_NUMBER}) (#${GITHUB_ISSUE_NUMBER})"
```

Make sure to insert the new line in alphabetical order if applicable.

## 4. Commit your changes

Add and commit the modified file:

```shell
git add documentation/provider/index.md
git commit -m "DOCS: Added requested provider ${PROVIDER_NAME} (#${GITHUB_ISSUE_NUMBER})"
```

## 5. Push and open a pull request

Push your changes to your fork and open a new pull request:

```shell
git push --no-verify
open "https://github.com/${GITHUB_FORK_REPO}/pull/new/docs/provider-request"
echo "Added ${PROVIDER_NAME} #${GITHUB_ISSUE_NUMBER} to the list of requested providers."
```

{% hint style="info" %}
**NOTE**: GitHub does not support pre-filling pull request titles or descriptions via URL parameters. The title will be auto-filled using your commit message. You can adjust it manually after opening the PR.
{% endhint %}

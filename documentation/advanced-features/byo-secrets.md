# Bring-Your-Own-Secrets for automated testing

Goal: Enable automated integration testing without accidentally leaking credentials (API keys and other secrets); at the same time permit everyone to automate their own tests without having to share their credentials.

The instructions in this document will enable automated tests to run in these situations:

* PR from a project member:
  * All officially supported providers plus many others too.
* PR from an external people:
  * Automated tests run for providers that don't require secrets, which is currently only `BIND`.
* PR on a fork of DNSControl:
  * The forker can set up secrets in their fork and only those providers with secrets will be tested. They can "set it and forget it" and all their future PRs will receive all the benefits of automated testing.

# Background: How GitHub Actions protects secrets

GitHub Actions has a secure [secrets storage system](https://docs.github.com/en/free-pro-team@latest/actions/reference/encrypted-secrets). Those secrets are available to GitHub Actions and are required for the integration tests to communicate with the various DNS providers that DNSControl supports.

For security reasons, those secrets are unavailable if the PR comes from outside the project (a forked repo).  This is a good thing.  If it didn't work that way, a third-party could write a PR that leaks the secrets without the owners of the project knowing.

The docs (and many blog posts) describe this as forked repos don't have access to secrets, and instead receive null strings. That's not actually what's happening.

Actually what happens is the secrets come from the forked repo.  Or, more precisely, the secrets offered to a PR come from the repo that the PR came from.  A PR from DNSControl's owners gets secrets from [github.com/DNSControl/dnscontrol's secret store](https://github.com/DNSControl/dnscontrol/settings/secrets/actions) but a PR from a fork, such as [https://github.com/TomOnTime/dnscontrol](https://github.com/TomOnTime/dnscontrol) gets its secrets from TomOnTime's secrets.

Our automated integration tests leverages this info to have tests only run if they have access to the secrets they will need.

# Which providers are selected for testing?

Tests are executed if the env variable`*_DOMAIN` exists where `*` is the name of the provider.  If the value is empty or unset, the test is skipped. For example, if a provider is called `FANCYDNS`, there must be a variable called `FANCYDNS_DOMAIN`.

# Bring your own secrets

This section describes how to add a provider to the "Actions" part of GitHub.

Step 1: Create a branch

Create a branch as you normally would to submit a PR to the project.

Step 2: Update `pr_integration_tests.yml`

{% hint style="info" %}
Edits to `pr_integration_tests.yml` may have already been done for you.
{% endhint %}

Edit `.github/workflows/pr_integration_tests.yml`

1. Add your providers `_DOMAIN` env variable:

* Add it to the `env` section of `integration-tests`.
* Please keep this list sorted alphabetically.

To find this section, search for `PROVIDER SECRET LIST`.

For example, the entry for BIND looks like:

{% code title=".github/workflows/pr_integration_tests.yml" %}
```
        BIND_DOMAIN: ${{ vars.BIND_DOMAIN }}
```
{% endcode %}

3. Add your providers other ENV variables:

Every provider requires different variables set to perform the integration tests.  The list of such variables is in `integrationTest/profiles.json`.

You've already added `*_DOMAIN` to `pr_integration_tests.yml`. Now we're going to add the remaining ones.

To find this section, search for `PROVIDER SECRET LIST`.

For example, the entry for CLOUDFLAREAPI looks like this:

{% code title=".github/workflows/pr_integration_tests.yml" %}
```
        CLOUDFLAREAPI_ACCOUNTID: ${{ secrets.CLOUDFLAREAPI_ACCOUNTID }}
        CLOUDFLAREAPI_TOKEN: ${{ secrets.CLOUDFLAREAPI_TOKEN }}
```
{% endcode %}

Step 3. Add the secrets to the repo.

The `*_DOMAIN` variable is stored as a "variable" while the others are stored as "secrets".

1. Go to Settings -> Secrets and variables -> Actions.

2. On the "Variables" tab, add `*_DOMAIN` with the name of a test domain. This domain must already exist in the account. The DNS records of the domain will be deleted, so please use a test domain or other disposable domain.

{% hint style="info" %}
For the main project, **variables** are added here: [https://github.com/DNSControl/dnscontrol/settings/variables/actions](https://github.com/DNSControl/dnscontrol/settings/variables/actions)
{% endhint %}

3. On the "Secrets" tab, add the other env variables.

{% hint style="info" %}
For the main project, **secrets** are added here: [https://github.com/DNSControl/dnscontrol/settings/secrets/actions](https://github.com/DNSControl/dnscontrol/settings/secrets/actions)
{% endhint %}

If you have forked the project, add these to the settings of that fork.

Step 4. Submit this PR like any other.

GitHub Actions should kick and and run the tests.

The tests will fail if a secret is wrong or missing.  It may take a few iterations to get everything working because... computers.

# Donate secrets to the project

The DNSControl project would like to have all providers automatically tested. However, we can't fund purchasing domains or maintaining credentials at every provider. Instead we depend on volunteers to maintain (and pay for) such accounts.

We recommend the domain be named `dnscontroltest-PROVIDER.com` (or similar) where PROVIDER is replaced by the name of your provider or an abbreviation. For example `dnscontroltest-r53.com` and `dnscontroltest-gcloud.com`.

When possible, use an OTE or free domain. Don't spend money if you don't have to. This isn't just to be thrifty! It avoids renewals and other hassles too. You'd be surprised at how many providers (such as Google and Azure) permit DNS zones to be created in your account without registering them.

For actual DNS domains, please select the "private registration" option if it is available. Otherwise you will get spam phones calls and emails. The phone calls will make you wish you didn't own a phone.

{% hint style="danger" %}
Some rules:

* The account/credentials should only access the test domain. Don't send your company's actual credentials and trust us to only touch the test domain. (this hasn't happened yet, thankfully!)
* Renew the domain in a timely manner. This may be monitoring an email inbox you don't normally monitor.
* Don't do anything that will get you in trouble with your employer, like charging it to your employer without permission. (this hasn't happened yet either, thankfully!)
{% endhint %}

Now that we've covered all that...

Create a new Github issue with a subject "Add PROVIDER to automated tests" where "PROVIDER" is the name of the provider. DO NOT SEND THE CREDENTIALS IN THE GITHUB ISSUE.  In the body write, "I understand the rules listed in byo-secrets.md and would like to volunteer to maintain the credentials and account."

To securely send the credentials to the project, please create a [SendSafely](https://www.sendsafely.com/) account (there is a free tier)
and use it to send them to tal at what exit dot org.  You'll hear back within a week.

Thank you for contributing credentials. The more providers we can test automatically with each PR, the better. It "shifts left" finding bugs and API changes and makes less work for everyone.

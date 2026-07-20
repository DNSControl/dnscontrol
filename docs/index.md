---
layout: default
title: DNSControl
---

<div class="row jumbotron">
    <div class="col-md-12">
        <div>
            <h1 class="hometitle">DNSControl</h1>
            <p class="lead">DNSControl is an <strong><a href="https://docs.dnscontrol.org/developer-info/opinions">opinionated</a></strong> platform for seamlessly managing your DNS configuration across any number of DNS hosts, both in the cloud or in your own infrastructure. It manages all of the domains for the Stack Overflow network, and can do the same for you!</p>
        </div>
    </div>
</div>

<div class="row text-center" style="padding-top: 75px;">
    <div class="col-md-4">
        <h3>Try It</h3>
        <p>Want to jump right in? Follow our
         <strong><a href="https://docs.dnscontrol.org/getting-started/getting-started">quick start tutorial</a></strong>
         on a new domain or
         <strong><a href="https://docs.dnscontrol.org/getting-started/migrating">migrate</a></strong>
         an existing one. Read the
         <strong><a href="https://docs.dnscontrol.org/language-reference/js">language spec</a></strong>
         for more info. You can also <strong><a href="#getting-started">view a list of all topics</a></strong>.
    </p>
    </div>

    <div class="col-md-4">
        <h3>Use It</h3>
        <p>Take advantage of the
         <strong><a href="#advanced-features">advanced features</a></strong>.
         Use macros and variables for easier updates.
         <!-- Optimize your SPF records. -->
         Upload your zones to
         <strong><a href="https://docs.dnscontrol.org/provider">multiple DNS providers</a></strong>.
    </p>
    </div>

    <div class="col-md-4">
        <h3>Get Involved</h3>
        <p>Join our
         <strong><a href="https://groups.google.com/forum/#!forum/dnscontrol-discuss">mailing list</a></strong>.
         We make it easy to contribute by using
         <strong><a href="https://github.com/DNSControl/dnscontrol">GitHub</a></strong>,
         you can make code changes with confidence thanks to extensive integration tests.
         The project is
         <strong><a href="https://everythingsysadmin.com/2017/08/go-get-up-to-speed.html">newbie-friendly</a></strong>
         so jump right in!
    </p>
    </div>
</div>

<div class="row" style="padding-top: 75px"><div class='col-md-4 col-md-offset-4'><h2 class="text-center feature-header">Features</h2></div></div>
<hr class="feature">

<div class="row">
    {% include feature.html text="Maintain your DNS data in a high-level DSL, with macros and variables for easier updates." img="biology.svg" %}
    {% include feature.html text="Super extensible! Plug-in architecture makes adding new DNS providers and Registrars easy!" img="light-bulb.svg" %}
    {% include feature.html text="Eliminate vendor lock-in. Switch DNS providers easily, any time, with full fidelity." img="group.svg" %}
    {% include feature.html text="Reduce points of failure: Easily maintain dual DNS providers and easily drop one that is down." img="layers.svg" %}
    {% include feature.html text="Supports 35+ DNS Providers including BIND, AWS Route 53, Google DNS, and name.com" img="cancel.svg" %}
    {% include feature.html text="Apply CI/CD principles to DNS: Unit-tests, system-tests, automated deployment." img="share.svg" %}
    {% include feature.html text="All the benefits of Git (or any VCS) for your DNS zone data. View history. Accept PRs." img="document.svg" %}
    {% include feature.html text="Optimize DNS with SPF optimizer. Detect too many lookups. Flatten includes." img="mail.svg" %}
    {% include feature.html text="Runs on Linux, Windows, Mac, or any operating system supported by Go." img="speech-bubble.svg" %}
    {% include feature.html text="Enable/disable Cloudflare proxying (the \"orange cloud\" button) directly from your DNSControl files." img="cloud-computing.svg" %}
    {% include feature.html text="Assign an IP address to a constant and use the variable name throughout the configuration. Need to change the IP address globally? Just change the variable and \"recompile.\"" img="compass.svg" %}
    {% include feature.html text="Keep similar domains in sync with transforms, macros, and variables." img="attachment.svg" %}
</div>

<hr class="feature">

<div class="container-fluid">
    <div class="row">
        <div class="col-md-4">
            <h2 id="getting-started">
                Getting Started
            </h2>
            <p>
                Information for new users and the curious.
            </p>

            <ul>
                <li>
                      <a href="https://docs.dnscontrol.org/getting-started/getting-started">Getting Started</a>: A walk-through of the basics
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/provider">Providers</a>: Which DNS providers are supported
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/getting-started/examples">Examples</a>: The DNSControl language by example
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/getting-started/migrating">Migrating</a>: Migrating zones to DNSControl
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/getting-started/typescript">TypeScript</a> (optional): Improve autocomplete and add type checking
                </li>
            </ul>

            <h2 id="commands">
                Commands
            </h2>
            <p>
                DNSControl sub-commands and options.
            </p>

            <ul>
                <li>
                     <a href="https://docs.dnscontrol.org/commands/creds-json">creds.json</a>: creds.json file format
                </li>
                <li>
                     <a href="https://docs.dnscontrol.org/commands/check-creds">check-creds</a>: Verify credentials
                </li>
                <li>
                     <a href="https://docs.dnscontrol.org/commands/get-zones">get-zones</a>: Query a provider for zone info
                </li>
            </ul>

        </div>
        <div class="col-md-4">
            <h2 id="reference">
                Reference
            </h2>
            <p>
                Language resources and procedures.
            </p>

            <ul>
                <li>
                    <a href="https://docs.dnscontrol.org/language-reference/js">Language Reference</a>: Description of the DNSControl language (DSL)
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/developer-info/alias">Aliases</a>: ALIAS/ANAME records
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/language-reference/domain-modifiers/spf_builder">SPF Optimizer</a>: Optimize your SPF records
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/language-reference/domain-modifiers/caa_builder">CAA Builder</a>: Build CAA records the easy way
                </li>
            </ul>
        </div>
        <div class="col-md-4">
            <h2 id="advanced-features">
                Advanced features
            </h2>
            <p>
                Take advantage of DNSControl's unique features.
            </p>
            <ul>
                <li>
                    <a href="https://docs.dnscontrol.org/language-reference/why-the-dot">Why CNAME/MX/NS targets require a trailing "dot"</a>
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/developer-info/unittests">Testing</a>: Unit Testing for you DNS Data
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/advanced-features/notifications">Notifications</a>: Web-hook for changes
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/advanced-features/code-tricks">Code Tricks</a>: Safely use macros and loops.
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/advanced-features/cli-variables">CLI variables</a>: Passing variables from CLI to JS
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/advanced-features/nameservers">Nameservers &amp; Delegation</a>: Many examples.
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/advanced-features/ci-cd-gitlab">Gitlab CI/CD example</a>.
                </li>
            </ul>
        </div>
    </div>
    <div class="row">
        <div class="col-md-12">
            <h2 id="developer-info">
                Developer Info
            </h2>
            <p>
                It is easy to add features and new providers to DNSControl. The code is very modular and easy to modify. There are extensive integration tests that make it easy to boldly make changes with confidence that you'll know if anything is broken. Our mailing list is friendly. Afraid to make your first PR? We'll gladly mentor you through the process. Many major code contributions have come from <a href="https://everythingsysadmin.com/2017/08/go-get-up-to-speed.html">first-time Go users</a>!
            </p>
            <ul>
                <li>
                    GitHub <a href="https://github.com/DNSControl/dnscontrol">DNSControl/dnscontrol</a>: Get the source!
                </li>
                <li>
                    Mailing list: <a href="https://groups.google.com/forum/#!forum/dnscontrol-discuss">dnscontrol-discuss</a>: The friendly best place to ask questions and propose new features
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/developer-info/bug-triage">Bug Triage</a>: How bugs are triaged
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/release/release-engineering">Release Engineering</a>: How to build and ship a release
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/developer-info/byo-secrets">Bring-Your-Own-Secrets</a>: Automate tests
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/developer-info/writing-providers">Step-by-Step Guide: Writing Providers</a>: How to write a DNS or Registrar Provider
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/developer-info/adding-new-rtypes">Step-by-Step Guide: Adding new DNS rtypes</a>: How to add a new DNS record type
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/developer-info/ordering">DNS reordering</a>: How DNSControl determines the order of the changes
                </li>
                <li>
                    <a href="https://docs.dnscontrol.org/developer-info/test-a-branch">Test a branch</a>: Instructions for testing DNSControl at a particular PR or branch
                </li>
            </ul>
        </div>
    </div>
</div>

<hr class="feature">

<p><small>Icons made by Freepik from <a href="https://www.flaticon.com">www.flaticon.com</a></small></p>

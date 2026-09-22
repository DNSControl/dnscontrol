<!--
## Before submiting a pull request

Please make sure you've run the following commands from the root directory.

    bin/generate-all.sh

(this runs commands like "go generate", fixes formatting, and so on)

## Pull request title

The pull request title becomes the commit message on main and must follow Conventional Commits (the "PR: Commitlint" check enforces this). Provider-specific changes use the scope "p/PROVIDERNAME". The subject starts with a lowercase letter.

Some examples:
* ci: add required GHA permissions for goreleaser
* docs: fix providers with "contributor support" table
* feat(p/ROUTE53): allow R53_ALIAS records to enable target health evaluation

More details can be found in CONTRIBUTING.md under "Pull request titles".
!-->

// Conventional Commits rules for PR titles.
//
// DNSControl squash-merges PRs, so the PR *title* becomes the commit
// message on main and is what goreleaser's changelog.groups (.goreleaser.yml)
// matches against. This config is linted in CI against the PR title, not
// against individual commits within the PR (see
// .github/workflows/pr_commitlint.yml).
//
// Provider-specific changes use a scope of "p/PROVIDER", e.g.:
//   fix(p/CLOUDFLAREAPI): correct TTL rounding
//   feat(p/ROUTE53): support alias records for NS
//
// Everything else uses one of the standard Conventional Commits types.

/** @type {import('@commitlint/types').UserConfig} */
module.exports = {
  extends: ['@commitlint/config-conventional'],
  parserPreset: {
    parserOpts: {
      // The default header pattern only allows a single word (\w+) as the
      // type. We widen it to also accept the literal types "BREAKING CHANGE"
      // and "BREAKING CHANGES", which contain a space.
      headerPattern: /^([a-zA-Z ]+)(?:\(([^)]+)\))?(!)?: (.+)$/,
      headerCorrespondence: ['type', 'scope', 'breaking', 'subject'],
    },
  },
  rules: {
    'type-enum': [
      2,
      'always',
      [
        'build',
        'chore',
        'ci',
        'docs',
        'feat',
        'fix',
        'perf',
        'refactor',
        'style',
        'test',
        'BREAKING CHANGE',
        'BREAKING CHANGES',
      ],
    ],
    // type-case can't be "always lower-case" since BREAKING CHANGE(S) is
    // uppercase by convention; type-enum above already constrains the set
    // of allowed values (case-sensitively), so the case rule is redundant.
    'type-case': [0],
    'scope-case': [0],
    'provider-scope-format': [2, 'always'],
  },
  plugins: [
    {
      rules: {
        // If a scope is given, it must be "p/PROVIDER" (provider-specific
        // change) or one of the config-conventional defaults are allowed too
        // (e.g. "deps" for dependency bumps). Everything else is free-form.
        'provider-scope-format': ({ scope }) => {
          if (!scope) {
            return [true];
          }
          if (/^p\/[A-Za-z0-9_.-]+$/.test(scope)) {
            return [true];
          }
          if (/^[a-z0-9_.-]+$/i.test(scope)) {
            return [true];
          }
          return [
            false,
            'scope must be a provider scope like "p/CLOUDFLAREAPI", or a simple word (e.g. "deps")',
          ];
        },
      },
    },
  ],
};

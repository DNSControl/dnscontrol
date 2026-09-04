#!/bin/bash
#
# bin/auto_release.sh -- kick off a DNSControl release.
#
# A release is simply "merge develop into main". This script opens (or reuses)
# the `develop -> main` pull request and turns on GitHub auto-merge. From there
# it is hands-off: the release gate (the full longtest integration suite) runs
# on the PR, and when it passes GitHub merges the PR automatically. That merge
# to `main` triggers .github/workflows/release_on_merge.yml, which picks the
# version with svu, tags it, and publishes via GoReleaser.
#
# You do NOT choose a version number, and there is no "Release vX.Y.Z" commit:
# the version is computed from the Conventional Commits on `develop` and lives
# only in the git tag.
#
# Usage:
#   bin/auto_release.sh [-y] [-w] [--skip-longtest]
#     -y, --yes           Do not prompt for confirmation.
#     -w, --watch         Watch the PR's checks after enabling auto-merge.
#     --skip-longtest     EMERGENCY: label the PR so the longtest gate is
#                         skipped. Use only when you must ship immediately.
#     -h, --help          Show this help.
#
# Requirements: git, and the GitHub CLI `gh` authenticated with write access.
# The repo must have "Allow auto-merge" enabled (Settings -> General).

set -euo pipefail

BASE="main"
HEAD="develop"
ASSUME_YES=0
WATCH=0
SKIP_LONGTEST=0

while [ $# -gt 0 ]; do
  case "$1" in
    -y|--yes)          ASSUME_YES=1 ;;
    -w|--watch)        WATCH=1 ;;
    --skip-longtest)   SKIP_LONGTEST=1 ;;
    -h|--help)         grep '^#' "$0" | sed -e 's/^#!.*//' -e 's/^# \{0,1\}//'; exit 0 ;;
    *)                 echo "Unknown option: $1" >&2; exit 2 ;;
  esac
  shift
done

command -v gh >/dev/null 2>&1 || { echo "ERROR: the GitHub CLI 'gh' is required." >&2; exit 1; }
gh auth status >/dev/null 2>&1 || { echo "ERROR: 'gh' is not authenticated (run: gh auth login)." >&2; exit 1; }

echo "Fetching origin..."
git fetch --quiet origin --tags

git show-ref --verify --quiet "refs/remotes/origin/$HEAD" || {
  echo "ERROR: origin/$HEAD does not exist. Create the '$HEAD' branch first." >&2; exit 1; }
git show-ref --verify --quiet "refs/remotes/origin/$BASE" || {
  echo "ERROR: origin/$BASE does not exist." >&2; exit 1; }

# Is there anything to release?
ahead="$(git rev-list --count "origin/$BASE..origin/$HEAD")"
if [ "$ahead" -eq 0 ]; then
  echo "Nothing to release: origin/$HEAD is not ahead of origin/$BASE."
  exit 0
fi

echo
echo "Commits on '$HEAD' not yet on '$BASE' that would trigger a version bump:"
if ! git log --no-merges --format='  %s' "origin/$BASE..origin/$HEAD" \
      | grep -Ei '^[[:space:]]*(feat|fix|perf)(\(|!|:)'; then
  echo "  (none matched feat/fix/perf -- svu may compute NO release)"
fi
echo
echo "Total commits ahead: $ahead"

# Best-effort: warn if the release automation is still disabled.
if enabled="$(gh variable list 2>/dev/null | awk '$1=="AUTO_RELEASE_ENABLED"{print $2}')" \
   && [ -n "$enabled" ] && [ "$enabled" != "true" ]; then
  echo
  echo "WARNING: repo variable AUTO_RELEASE_ENABLED='$enabled' (not 'true')."
  echo "         The PR will merge, but NO release is cut until it is 'true'."
fi

if [ "$ASSUME_YES" -ne 1 ]; then
  echo
  printf "Open / enable auto-merge for the %s -> %s release PR? [y/N] " "$HEAD" "$BASE"
  read -r reply
  case "$reply" in
    y|Y|yes|YES) ;;
    *) echo "Aborted."; exit 0 ;;
  esac
fi

# Create the PR if one does not already exist for this branch.
if pr_url="$(gh pr view "$HEAD" --json url --jq .url 2>/dev/null)" && [ -n "$pr_url" ]; then
  echo "Reusing existing PR: $pr_url"
else
  pr_url="$(gh pr create --base "$BASE" --head "$HEAD" \
    --title "Release: merge $HEAD into $BASE" \
    --body "Automated release PR opened by \`bin/auto_release.sh\`. When the release gate passes this auto-merges (as a merge commit) and a release is cut; svu picks the version.")"
  echo "Opened PR: $pr_url"
fi

if [ "$SKIP_LONGTEST" -eq 1 ]; then
  echo "EMERGENCY: adding 'skip-longtest' label (the integration gate will be bypassed)."
  gh pr edit "$pr_url" --add-label skip-longtest
fi

# Enable auto-merge with a MERGE COMMIT. Never squash: squashing would collapse
# develop's Conventional Commits into one and break both the version bump and
# the changelog.
gh pr merge "$pr_url" --auto --merge

echo
echo "Auto-merge enabled. When the required checks pass, GitHub will merge"
echo "$HEAD -> $BASE and the release will publish automatically."
echo "(If branch protection on '$BASE' requires an approval, the PR waits for it.)"

if [ "$WATCH" -eq 1 ]; then
  echo
  echo "Watching checks (Ctrl-C to stop; stopping does not cancel the release)..."
  gh pr checks "$pr_url" --watch || true
fi

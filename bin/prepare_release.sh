#!/bin/bash

# This script helps prepare new releases. It creates a branch
# called "prep_release" which contains any last changes we
# tend to make before a release: update dependencies, go generate, go fix.

# The script tries to match the common (human) workflow:
# Run this in the "main" branch and it will switch to the branch
#     "prep_release", creating it if needed. It branches from "origin/main",
#     thus will ignore any changes, even if they were committed to "main".
# Run this in "prep_release" branch and it will do its work in the existing
#     branch. It will retain any uncommitted changes. Thus, allowing you to
#     run it many times, making changes along the way.
# It will refuse to run from any other branch.

# What is the current git branch?
current_branch=$(git rev-parse --abbrev-ref HEAD)

# If current branch is not "prep_release"...
if [ "$current_branch" != "prep_release" ]; then

    # If branch "prep_release" exists...
    if git show-ref --verify --quiet refs/heads/prep_release; then

        # Exit. We don't want to accidentally wreck what is there.
        echo "Error: Branch 'prep_release' already exists. Please switch to it or delete it first."
        echo "Delete branch: git branch -D prep_release"
        echo "or... Proceed: git checkout prep_release"
        exit 1
    fi

    # It doesn't exit. Create it.

    # Make sure we have the latest 'main' so that when we branch, we get the
    # latest code.
    git fetch origin main
    git reset --hard origin/main

fi

# What is the current git branch?
current_branch=$(git rev-parse --abbrev-ref HEAD)

# Change to the prep_release branch
if [ "$current_branch" = "prep_release" ]; then
    git checkout prep_release
else
    git checkout -b prep_release
fi

# Run go-mod-upgrade to update dependencies
go install github.com/oligot/go-mod-upgrade@latest
go-mod-upgrade
go mod tidy
git commit -m "chore: Update dependencies" go.sum go.mod

# Regenerate
bin/generate-all.sh
git status
echo "SUGGESTION:" 'git commit -m "chore: generate-all.sh" -a'

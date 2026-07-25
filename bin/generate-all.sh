#!/bin/sh

echo ========== go fmt
go fmt ./...

if [ -x node_modules/.bin/prettier ]; then
  echo ========== prettier
  node_modules/.bin/prettier --write pkg/js/helpers.js
fi

echo ========== bin/fmtjson
bin/fmtjson $(find . -path ./.vscode -prune -o -type f -name "*.json" ! -name "package-lock.json" -print)

# dnsconfig.js-compatible files:
echo ========== fmt parse_tests
for i in pkg/js/parse_tests/*.js ; do dnscontrol fmt -i $i -o $i ; done

echo ========== go generate
go generate ./...

echo ========== go mod tidy
go mod tidy

echo ========== go fix ./...
go fix ./...

echo ==== Running golangci-lint run ./...
if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run ./...
  else
  echo "golangci-lint not found, skipping"
fi

echo ==== Running staticcheck ./...
if command -v staticcheck >/dev/null 2>&1; then
  staticcheck ./...
else
  echo "staticcheck not found, skipping"
fi

# NOTE:
# This is one way to make sure Unicode chars haven't crept into pkg/js/parse_tests
# pcre2grep --color='auto' -n '[^\x00-\x7F]' $(find * . -type f -name \*.json)


#!/bin/sh

set -e

cd $(git rev-parse --show-toplevel)
cd pkg/privatetypes
go run types_generate.go && goimports -w *.go */*.go

#!/bin/sh

bin/fmtjson $(find \
	dist \
	commands/test_data \
        integrationTest pkg \
	documentation/assets  \
	-type f -name "*.json" \
	! -name "package-lock.json" \
	-print)

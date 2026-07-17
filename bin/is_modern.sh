#!/bin/sh

cd "${1:-.}"

echo '========== RecordConfig{'
grep --include='*.go' -r -F 'RecordConfig{' *
echo '========== PopulateFromString{'
grep --include='*.go' -r -F 'PopulateFromString' *
echo '========== SetTarget'
grep --include='*.go' -r -F 'SetTarget' *
echo '========== GetTarget'
grep --include='*.go' -r -F 'GetTarget' *

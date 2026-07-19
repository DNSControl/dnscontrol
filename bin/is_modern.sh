#!/bin/bash

cd "${1:-.}"
if [[ $? -ne 0 ]] ; then
    exit 1
fi

echo '========== DomainConfig{'
grep --color --include='*.go' -r -F 'DomainConfig{' *

echo '========== RecordConfig{'
grep --color --include='*.go' -r -F 'RecordConfig{' *

echo '========== PopulateFromString{'
grep --color --include='*.go' -r -F 'PopulateFromString' *

echo '========== AddOrigin('
grep --color --include='*.go' -r -F 'AddOrigin(' *

echo '========== SetTarget'
grep --color --include='*.go' -r -E 'GetTargetCombinedFunc\(|GetTargetCombined\(|GetTargetRFC1035Quoted\('

echo '========== GetTarget'
grep --color --include='*.go' -r -E 'SetTargetCAA\(|SetTargetCAAStrings\(|SetTargetCAAString\(|SetTargetDNSKEYString\(|SetTargetDSString\(|SetTargetLOCString\(|SetTargetMX\(|SetTargetMXString\(|SetTargetNAPTR\(|SetTargetNAPTRString\(|SetTargetSMIMEA\(|SetTargetSOA\(|SetTargetSRV\(|SetTargetSRVPriorityString\(|SetTargetSRVString\(|SetTargetSSHFP\(|SetTargetSSHFPStrings\(|SetTargetSSHFPString\(|SetTargetSVCBString\(|SetTargetTLSA\(|SetTargetTLSAString\('

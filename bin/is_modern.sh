#!/bin/bash

cd "${1:-.}"
if [[ $? -ne 0 ]] ; then
    exit 1
fi

echo '========== Step 1. DomainConfig{'
grep --color --include='*.go' -r -F 'DomainConfig{' *

echo '========== Step 2. RecordConfig{'
grep --color --include='*.go' -r -F 'RecordConfig{' *

echo '========== Step 3. PopulateFromString{'
grep --color --include='*.go' -r -F 'PopulateFromString' *

echo '========== Step 3a.dnsv1'
grep --color --include='*.go' -r -F -e github.com/miekg/dns *


echo '========== Step 4. AddOrigin('
grep --color --include='*.go' -r -F 'AddOrigin(' *

echo '========== Step 5. TrimDomainName('
grep --color --include='*.go' -r -F 'TrimDomainName(' *

echo '========== Step 6. SetTarget'
grep --color --include='*.go' -r -E 'GetTargetCombinedFunc\(|GetTargetCombined\(|GetTargetRFC1035Quoted\('

echo '========== Step 7. GetTarget'
grep --color --include='*.go' -r -E 'SetTargetCAA\(|SetTargetCAAStrings\(|SetTargetCAAString\(|SetTargetDNSKEYString\(|SetTargetDSString\(|SetTargetLOCString\(|SetTargetMX\(|SetTargetMXString\(|SetTargetNAPTR\(|SetTargetNAPTRString\(|SetTargetSMIMEA\(|SetTargetSOA\(|SetTargetSRV\(|SetTargetSRVPriorityString\(|SetTargetSRVString\(|SetTargetSSHFP\(|SetTargetSSHFPStrings\(|SetTargetSSHFPString\(|SetTargetSVCBString\(|SetTargetTLSA\(|SetTargetTLSAString\('

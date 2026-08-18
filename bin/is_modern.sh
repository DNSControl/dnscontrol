#!/bin/bash

cd "${1:-.}"
if [[ $? -ne 0 ]] ; then
    exit 1
fi

echo '========== Step 1. DomainConfig{'
grep -n --include='*.go' -r -F -e 'DomainConfig{' | grep -v -F '[]*models.DomainConfig'  | grep --color 'DomainConfig{'

echo '========== Step 2. RecordConfig{'
grep -n --color --include='*.go' -r -F 'RecordConfig{' *

echo '========== Step 3. PopulateFromString{'
grep -n --color --include='*.go' -r -F 'PopulateFromString' *

echo '========== Step 3a.dnsv1'
grep -n --color --include='*.go' -r -F -e github.com/miekg/dns *


echo '========== Step 4. AddOrigin('
grep -n --color --include='*.go' -r -F 'AddOrigin(' *

echo '========== Step 5. TrimDomainName('
grep -n --color --include='*.go' -r -F 'TrimDomainName(' *

echo '========== Step 6. SetTarget'
grep -n --color --include='*.go' -r -E 'GetTargetCombinedFunc\(|GetTargetCombined\(|GetTargetRFC1035Quoted\(' *

echo '========== Step 7. GetTarget'
grep -n --color --include='*.go' -r -E 'SetTargetCAA\(|SetTargetCAAStrings\(|SetTargetCAAString\(|SetTargetDNSKEYString\(|SetTargetDSString\(|SetTargetLOCString\(|SetTargetMX\(|SetTargetMXString\(|SetTargetNAPTR\(|SetTargetNAPTRString\(|SetTargetSMIMEA\(|SetTargetSOA\(|SetTargetSRV\(|SetTargetSRVPriorityString\(|SetTargetSRVString\(|SetTargetSSHFP\(|SetTargetSSHFPStrings\(|SetTargetSSHFPString\(|SetTargetSVCBString\(|SetTargetTLSA\(|SetTargetTLSAString\(' *

echo '========== Step 8. Old Fields'
regex='\.(MxPreference|SrvPriority|SrvWeight|SrvPort|CaaTag|CaaFlag|DsKeyTag|DsAlgorithm|DsDigestType|DsDigest|DnskeyFlags|DnskeyProtocol|DnskeyAlgorithm|DnskeyPublicKey|LocVersion|LocSize|LocHorizPre|LocVertPre|LocLatitude|LocLongitude|LocAltitude|NaptrOrder|NaptrPreference|NaptrFlags|NaptrService|NaptrRegexp|SmimeaUsage|SmimeaSelector|SmimeaMatchingType|SshfpAlgorithm|SshfpFingerprint|SoaMbox|SoaSerial|SoaRefresh|SoaRetry|SoaExpire|SoaMinttl|SvcPriority|SvcParams|TlsaUsage|TlsaSelector|TlsaMatchingType)'
grep -n --include='*.go' --exclude=populatelegacy.go -r -E "$regex" * | grep -v 'ignore:legacyfield' | grep --color -E "$regex"

echo '========== Step 9. GetTargetField'
grep -n --color --include='*.go' -r -E 'GetTargetField\(' *

# echo '========== Step 10. Should Be Metadata Fields'
# grep -n --color --include='*.go' -r -E '\.(LuaType|R53Alias|AzureAlias|AnswerType|UnknownTypeName)' *

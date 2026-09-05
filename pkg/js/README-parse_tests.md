# Parse Tests

- [Parse Tests](#parse-tests)
  - [Fix formatting](#fix-formatting)
  - [Copy actuals to expected](#copy-actuals-to-expected)

The `parse_tests` directory contains test cases for `js_test.go`.  `js_test.go`
scans for files named `DDD-*.js` where `DDD` is a three-digit number.

- `parse_tests/001-basic.js`  -- The dnsconfig.js file.
- `parse_tests/001-basic.json` -- The EXPECTED output of "print-ir" for the `.js` file.
- `parse_tests/001-basic.json.ACTUAL` -- The ACTUAL output of "print-ir" for the `.js` file (not saved in git)
- `parse_tests/001-basic/foo.com.zone` -- Zonefiles from the domains mentioned in dnsconfig.js

NOTE: The zonefiles are only tested if a matching `DDD-name/DOMAINNAME.zone` file exists.

Any files committed to Git should be in standard format.

## Fix formatting

Fix the `.js` formatting:

```shell
cd parse_tests
for i in *.js ; do echo ========== $i ; dnscontrol fmt -i $i -o $i ; done
```

Fix the `.json` formatting:

```shell
cd parse_tests
fmtjson *.json *.json.ACTUAL
```

## Copy actuals to expected

Back-port the ACTUAL results to the expected results:

(This is dangerous. You may be committing buggy results to the "expected" files. Carefully inspect the resulting PR.)

```shell
find . -type f -name \*.ACTUAL -print -delete
go test -count=1 ./...
cd parse_tests
fmtjson *.json *.json.ACTUAL
for i in *.ACTUAL ; do f=$(basename $i .ACTUAL) ; cp $i $f ; done
```

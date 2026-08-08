#!/bin/sh

# Run fmtjson on "all the directories we want".
# Note: Never fmt the providers/*/test_data files.

bin/fmtjson $(\
  find \
      ./commands/test_data \
      ./commands/testdata/init \
      ./documentation/assets \
      ./integrationTest \
      ./pkg/js/parse_tests \
      ./pkg/spflib \
          -name .claude -prune -o \
          -type f  \
          -name "*.json"  \
          ! -name "package-lock.json"  \
          -print \
  )

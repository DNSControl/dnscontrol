#!/bin/bash

set -e

SRC="Adguardhome_AAAA_Passthrough"
SRC_UC="$( echo "${SRC}" | tr a-z A-Z )"
SRC_LC="$( echo "${SRC}" | tr A-Z a-z )"
SRC_VAR="$( echo "${SRC}" | tr a-z A-Z | tr -d _ )"
SRC_Func="$( echo "${SRC}" | tr -d _ )"

DEST="$1" ; shift
DEST_UC="$( echo "${DEST}" | tr a-z A-Z )"
DEST_LC="$( echo "${DEST}" | tr A-Z a-z )"
DEST_VAR="$( echo "${DEST}" | tr a-z A-Z | tr -d _ )"
DEST_Func="$( echo "${DEST}" | tr -d _ )"

echo "SRC_VAR=$SRC_VAR"  
echo "DEST_VAR=$DEST_VAR"

cp "t_${SRC_LC}.go"              "t_${DEST_LC}.go"
cp "t_${SRC_LC}_test.go"         "t_${DEST_LC}_test.go"
cp "rdata/rdata_${SRC_LC}.go"    "rdata/rdata_${DEST_LC}.go"

for i in "t_${DEST_LC}.go" "t_${DEST_LC}_test.go" "rdata/rdata_${DEST_LC}.go" ; do
	sed -i.bak \
		-e "s@${SRC_UC}@${DEST_UC}@g"  	\
		-e "s@${SRC_VAR}@${DEST_VAR}@g" \
		-e "s@${SRC_Func}@${DEST_Func}@g" \
		-e "s@Test${SRC}@Test${DEST_Func}@g" \
			"$i"
	rm "$i".bak
done

num=$(echo 1 + $(grep -h  'const Type' *.go | awk '{ print $NF }'  |sort | tail -1) | bc)
echo "Codepoint: $num"
sed -i.bak -e "s/const Type.*/const Type${DEST_VAR} = $num/g" t_"${DEST_LC}.go"
rm "t_${DEST_LC}.go.bak"
grep -E "^const Type${SRC_VAR}" "t_${DEST_LC}.go"

echo '                case "'"${DEST_UC}"'":
                        rc.RDATA = privatetypesrdata.'"${DEST_VAR}"'{}' | pbcopy
vi +/Incomplete ../../models/fixhack.go

echo "vi ../../models/fixhack.go"
echo "../../integrationTest/integration_test.go"

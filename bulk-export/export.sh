#!/bin/sh
set -e
PAGE_SIZE=3000
LAST_ID=0
PREFIX=$$
while :
do
  OUTPUT=export-${PREFIX}-${LAST_ID}.json
  # the nested quotes are required, as the first pair (doubles) is "eaten" by the shell
  psql -t  \
    -v snapshot_timestamp="'2020-05-05 00:00:00.000'" \
    -v id_offset=${LAST_ID}  \
    -v page_size=${PAGE_SIZE} < export.sql > ${OUTPUT}
  COUNT=$(jq '.|length' < ${OUTPUT})
  if [ "${COUNT}" -ne "${PAGE_SIZE}" ]; # we got the last page
  then
    break
  fi
  LAST_ID=$(jq '.[-1].id'< ${OUTPUT})
  echo "."
done
echo "."
# combine the arrays -s, add, remove id as it is not part of schema
jq -s 'add|del(.[].id)' export-${PREFIX}-*.json > export-${PREFIX}.json
rm export-${PREFIX}-*.json
echo export-${PREFIX}.json

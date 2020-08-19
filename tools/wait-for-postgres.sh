#!/bin/sh
container=${1:-postgres}
tries=10
while [ $tries -gt 0 ]; do
   sleep 1
   if docker exec -u postgres "${container}" pg_isready -U postgres ;
   then
      exit 0
   else
     : $((tries=tries-1))
   fi
   echo $tries
done
echo "All tries exceeded. Giving up." 1>&1
exit 1
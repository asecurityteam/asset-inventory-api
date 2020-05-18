# Bulk export of resource assignments for point in time
## Summary
The [shell script](export.sh) and [SQL file](export.sql) allow to export all the resources 
and assignments for the specified timestamp utilizing new database schema.

The export runs manually via command line using connection to the Postgres database. 
This is a __STOPGAP__ solution until the bulk export endpoint `/v1/cloud/asset` is updated to use the new SQL schema.

The resulting JSON dump file has the format identical to the one defined in `#/components/schemas/BulkCloudAssets` in the 
[API description](../api.yaml)

## Using it
### Pre-requisites
* PSQL command line client
* `jq` utility
* Read access to Asset Inventory database or replica

### Caveats
* As this is a stopgap, the query performance is not most optimal (no indices for several things), json conversion on SQL server side takes tons of CPU/RAM.
* Please test in controlled environment before running this in production. Consider using separate disconnected replica.
* JSON wrangling via `jq` could definitely benefit by replacement with better validation and error handling. 

### Performance 
Around 4 seconds per page of 3K results with RDS, output ~14Mb of JSON. Definitely YMMV.

### Execution steps
* Set the standard PostgreSQL connection environment variables to access the DB.
* Change the `snapshot_timestamp` in the [export script](export.sh) (NB - nested quoting is required).
* Run the script. It prints a dot '.' after processing each page of results.
* When export completes, the name of resulting dump will be printed


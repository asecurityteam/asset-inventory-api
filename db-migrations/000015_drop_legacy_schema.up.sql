-- WARNING!!! this is not possible to roll back w/o back-fill of data from new schema or some other source
-- no transaction as it would be supermassive
DROP TABLE IF EXISTS aws_events_ips_hostnames; -- this automatically gets rid of partitions
DROP TABLE IF EXISTS aws_resources;
DROP TABLE IF EXISTS aws_ips;
DROP TABLE IF EXISTS aws_hostnames;
DROP TABLE IF EXISTS partitions;
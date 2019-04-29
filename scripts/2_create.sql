-- Package storage implements the database access layer.  The underlying database
-- is Postgres, and the tables are defined as such:

CREATE TABLE IF NOT EXISTS aws_resources
(
    id VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL,
    region VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    meta JSONB
);

-- We use these simple tables to preserve uniqueness and so we can add columns for additional
-- metadata when needed without polluting the aws_events_ips_hostnames star table:

CREATE TABLE IF NOT EXISTS aws_ips
(
    ip INET PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS aws_hostnames
(
    hostname VARCHAR PRIMARY KEY
);

-- Notice "PARTITION BY" below.  We're using built-in partitioning from Postgres 10+
-- See https://blog.timescale.com/scaling-partitioning-data-postgresql-10-explained-cd48a712a9a1/
-- Postgres 11 has many updates for partitioning (see https://pgdash.io/blog/partition-postgres-11.html)
-- that we are taking advantage of, such as automatic index propagation to child tables.

CREATE TABLE IF NOT EXISTS aws_events_ips_hostnames
(
    ts TIMESTAMP NOT NULL,
    is_public BOOLEAN NOT NULL,
    is_join BOOLEAN NOT NULL,
    aws_resources_id VARCHAR NOT NULL,
    FOREIGN KEY (aws_resources_id) REFERENCES aws_resources (id),
    aws_ips_ip INET NOT NULL,
    FOREIGN KEY (aws_ips_ip) REFERENCES aws_ips (ip),
    aws_hostnames_hostname VARCHAR,
    FOREIGN KEY (aws_hostnames_hostname) REFERENCES aws_hostnames (hostname)
)
PARTITION BY
    RANGE
(
        ts
);

-- And we'll make sure there's an index right away that Postgres 11 will automatically propagate to child tables:
-- (We use aws_ips_ip and aws_hostnames_hostname and timestamp as it is expected these will be the most frequently used in queries)

CREATE INDEX
IF NOT EXISTS aws_events_ips_hostnames_aws_ips_ip_ts_idx ON aws_events_ips_hostnames USING BTREE
(aws_ips_ip, ts);

CREATE INDEX
IF NOT EXISTS aws_events_ips_hostnames_aws_hostnames_hostname_ts_idx ON aws_events_ips_hostnames USING BTREE
(aws_hostnames_hostname, ts);

-- Also, some good advice to follow:  https://www.vividcortex.com/blog/2015/09/22/common-pitfalls-go/

-- The underlying code must be careful to create new partition tables when necessary.  Future updates to
-- the implementation where we want to track more asset types and values would require creation of new
-- tables, done by either a database admin or here in code, perhaps in the Init function.
--- WARNING this is designed to run outside any transaction, so no BEGIN is present
--- because `concurrently` (that avoids table write lock) is not compatible with DDL transactions
--- This schema change is to exclusively be run in background
create index concurrently if not exists idx_aws_resource_id on aws_private_ip_assignment (aws_resource_id);
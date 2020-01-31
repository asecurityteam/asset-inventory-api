BEGIN;
-- NB the order is important because of foreign keys!
drop table if exists aws_public_ip_assignment;
drop table if exists aws_private_ip_assignment;
drop table if exists aws_region;
drop table if exists aws_account;
drop table if exists aws_resource_type;
drop table if exists aws_resource;
COMMIT;

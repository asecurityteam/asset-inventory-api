-- Removing(roll-back) aws_resource_relationship table
BEGIN;

DROP TABLE IF EXISTS aws_resource_relationship CASCADE;

COMMIT;
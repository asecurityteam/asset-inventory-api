-- Removing indexes on aws_resource_relationship
BEGIN;

DROP INDEX CONCURRENTLY IF EXISTS aws_resource_relationship_idx_no_after;
DROP INDEX CONCURRENTLY IF EXISTS aws_resource_relationship_idx_no_before;

COMMIT;

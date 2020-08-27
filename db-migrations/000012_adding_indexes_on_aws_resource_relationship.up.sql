-- Adding indexes on aws_resource_relationship
BEGIN;

CREATE UNIQUE INDEX IF NOT EXISTS aws_resource_relationship_idx_no_after ON aws_resource_relationship (not_before, arn_id, related_arn_id) WHERE not_after IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS aws_resource_relationship_idx_no_before ON aws_resource_relationship (not_after, arn_id, related_arn_id) WHERE not_before IS NULL;

COMMIT;

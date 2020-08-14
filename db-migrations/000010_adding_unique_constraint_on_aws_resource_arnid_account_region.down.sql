-- Removing unique constraint on aws_resource table for columns arn_id, aws_account_id, aws_region_id
BEGIN;

ALTER TABLE aws_resource
    DROP CONSTRAINT IF EXISTS arn_account_region_id_unique;

ALTER TABLE aws_resource
    ADD CONSTRAINT aws_resource_arn_id_key UNIQUE (arn_id);

COMMIT;
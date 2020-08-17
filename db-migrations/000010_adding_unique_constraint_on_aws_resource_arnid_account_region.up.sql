-- Adding unique constraint on aws_resource table for columns arn_id, aws_account_id, aws_region_id
BEGIN;

ALTER TABLE aws_resource
    DROP CONSTRAINT IF EXISTS aws_resource_arn_id_key;

ALTER TABLE aws_resource
    ADD CONSTRAINT arn_account_region_id_unique UNIQUE (arn_id, aws_account_id, aws_region_id);

COMMIT;
-- schema changes to support dual writes
BEGIN;

DROP FUNCTION get_resource_by_arn_id(character varying, timestamp without time zone);
DROP FUNCTION get_owner_and_champions_by_account_id(integer);
DROP FUNCTION get_resource_by_hostname(character varying, timestamp without time zone);
DROP FUNCTION get_resource_by_private_ip(inet, timestamp without time zone);
DROP FUNCTION get_resource_by_public_ip(inet, timestamp without time zone);

COMMIT;

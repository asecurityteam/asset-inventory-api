-- roll-back of schema changes for account owners and champions
BEGIN;

DROP TABLE IF EXISTS person CASCADE;
DROP TABLE IF EXISTS account_champion CASCADE;
DROP TABLE IF EXISTS account_owner CASCADE;

COMMIT;

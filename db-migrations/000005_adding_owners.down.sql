-- roll-back of schema changes for account owners and champions
BEGIN;

DROP TABLE IF EXISTS person;
DROP TABLE IF EXISTS account_champion;
DROP TABLE IF EXISTS account_owner;

COMMIT;

-- schema changes for account owners and champions
BEGIN;
create table champions(
    id serial primary key,
    champion varchar not null
)

create table owners(
    id serial primary key,
    owner varchar not null,
    foreign key(aws_account_id) references aws_account (id)
)

alter table aws_account
-- accounts may or may not have champions associated with them, so we may not need to enforce a foreign key restraint
add column champion_id serial null unique

COMMIT;

-- schema changes for account owners and champions
BEGIN;

create table person(
    id serial primary key,
    login varchar unique not null,
    email varchar unique not null,
    name varchar not null,
    valid boolean not null
)

create table champions(
    foreign key(people_id) references person (id),
    foreign key(aws_account_id) references aws_account (id)
)

create table aws_accounts_owners(
    foreign key(people_id) references person (id),
    foreign key(aws_account_id) references aws_account (id) unique
)

COMMIT;

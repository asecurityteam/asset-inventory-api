-- schema changes for account owners and champions
BEGIN;

create table person(
    id serial primary key,
    login varchar unique not null,
    email varchar unique not null,
    name varchar not null,
    valid boolean not null
);

create table account_champion(
    person_id serial,
    aws_account_id integer,
    foreign key(person_id) references person (id),
    foreign key(aws_account_id) references aws_account (id),
    unique(person_id, aws_account_id)
);

create table account_owner(
    person_id serial,
    aws_account_id integer,
    foreign key(person_id) references person (id),
    foreign key(aws_account_id) references aws_account (id),
    unique (aws_account_id)
);

COMMIT;

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
    foreign key(people_id) references person (id),
    foreign key(aws_account_id) references aws_account (id),
    constraint account_champ_unique unique(people_id, aws_account_id)
);

create table account_owner(
    foreign key(people_id) references person (id),
    foreign key(aws_account_id) references aws_account (id),
    constraint account_unique unique (aws_account_id)
);

COMMIT;

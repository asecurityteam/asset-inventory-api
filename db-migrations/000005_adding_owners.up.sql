-- schema changes for account owners and champions
BEGIN;

create table person(
    id serial primary key,
    login varchar not null,
    email varchar not null,
    name varchar not null,
    valid boolean
)

create table champions(
    foreign key(people_id) references person (id),
    foreign key(aws_account_id) references aws_account (id)
)

create table owner(
    foreign key(people_id) references person (id),
    foreign key(aws_account_id) references aws_account (id)
)

COMMIT;

-- Adding aws_resource_relationship table
BEGIN;

create table if not exists aws_resource_relationship(
    id                serial primary key,
    arn_id            varchar not null,
    related_arn_id    varchar not null,
    not_before        timestamp,
    not_after         timestamp
);

COMMIT;
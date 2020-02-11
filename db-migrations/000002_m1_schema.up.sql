BEGIN;
create table aws_region
(
    id     serial primary key,
    region varchar not null unique
);

create table aws_account
(
    id      serial primary key,
    account varchar not null unique
);

create table aws_resource_type
(
    id            serial primary key,
    resource_type varchar not null unique
);

create table aws_resource
(
    id                   bigserial primary key,
    arn_id               varchar not null unique, /* nb, this is NOT full arn, it is resource ID like i-5325235 as ARN is redundant and can be reconstructed */
    aws_account_id       int     not null, /* the account, type and region are immutable, so the lookup tables are referenced as the part of resource, not event */
    foreign key (aws_account_id) references aws_account (id),
    aws_region_id        int     not null,
    foreign key (aws_region_id) references aws_region (id),
    aws_resource_type_id int     not null,
    foreign key (aws_resource_type_id) references aws_resource_type (id),
    meta                 JSONB /* this is not optimal and is not correct as tags can change and changes need tracking */
);


create table aws_public_ip_assignment
(
    id              bigserial primary key,
    not_before      timestamp not null,
    not_after       timestamp,
    public_ip       inet      not null,
    aws_hostname    varchar   not null,
    aws_resource_id bigint    not null,
    foreign key (aws_resource_id) references aws_resource (id)
);

create unique index aws_public_ip_assignment_idx_no_after on aws_public_ip_assignment (not_before, public_ip, aws_resource_id) where not_after is null;

create unique index aws_public_ip_assignment_idx_no_before on aws_public_ip_assignment (not_after, public_ip, aws_resource_id) where not_before is null;

create index idx_public_ip on aws_public_ip_assignment (public_ip);

create index idx_aws_hostname on aws_public_ip_assignment (aws_hostname);

create table aws_private_ip_assignment
(
    id              bigserial primary key,
    not_before      timestamp not null,
    not_after       timestamp,
    private_ip      inet      not null,
    aws_resource_id int       not null,
    foreign key (aws_resource_id) references aws_resource (id)
);

create unique index aws_private_ip_assignment_idx_no_after on aws_private_ip_assignment (not_before, private_ip, aws_resource_id) where not_after is null;

create unique index aws_private_ip_assignment_idx_no_before on aws_private_ip_assignment (not_after, private_ip, aws_resource_id) where not_before is null;

create index idx_private_ip on aws_private_ip_assignment (private_ip);
COMMIT;

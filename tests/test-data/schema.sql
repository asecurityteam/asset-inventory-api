--
-- PostgreSQL database dump
--

-- Dumped from database version 11.2 (Debian 11.2-1.pgdg90+1)
-- Dumped by pg_dump version 12.3

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: account_champion; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.account_champion (
    person_id integer NOT NULL,
    aws_account_id integer
);


ALTER TABLE public.account_champion OWNER TO "user";

--
-- Name: account_champion_person_id_seq; Type: SEQUENCE; Schema: public; Owner: user
--

CREATE SEQUENCE public.account_champion_person_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.account_champion_person_id_seq OWNER TO "user";

--
-- Name: account_champion_person_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: user
--

ALTER SEQUENCE public.account_champion_person_id_seq OWNED BY public.account_champion.person_id;


--
-- Name: account_owner; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.account_owner (
    person_id integer NOT NULL,
    aws_account_id integer
);


ALTER TABLE public.account_owner OWNER TO "user";

--
-- Name: account_owner_person_id_seq; Type: SEQUENCE; Schema: public; Owner: user
--

CREATE SEQUENCE public.account_owner_person_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.account_owner_person_id_seq OWNER TO "user";

--
-- Name: account_owner_person_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: user
--

ALTER SEQUENCE public.account_owner_person_id_seq OWNED BY public.account_owner.person_id;


--
-- Name: aws_account; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_account (
    id integer NOT NULL,
    account character varying NOT NULL
);


ALTER TABLE public.aws_account OWNER TO "user";

--
-- Name: aws_account_id_seq; Type: SEQUENCE; Schema: public; Owner: user
--

CREATE SEQUENCE public.aws_account_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.aws_account_id_seq OWNER TO "user";

--
-- Name: aws_account_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: user
--

ALTER SEQUENCE public.aws_account_id_seq OWNED BY public.aws_account.id;


--
-- Name: aws_events_ips_hostnames; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_events_ips_hostnames (
    ts timestamp without time zone NOT NULL,
    is_public boolean NOT NULL,
    is_join boolean NOT NULL,
    aws_resources_id character varying NOT NULL,
    aws_ips_ip inet NOT NULL,
    aws_hostnames_hostname character varying
)
PARTITION BY RANGE (ts);


ALTER TABLE public.aws_events_ips_hostnames OWNER TO "user";

--
-- Name: aws_events_ips_hostnames_2019_05_07to2019_05_21; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_events_ips_hostnames_2019_05_07to2019_05_21 (
    ts timestamp without time zone NOT NULL,
    is_public boolean NOT NULL,
    is_join boolean NOT NULL,
    aws_resources_id character varying NOT NULL,
    aws_ips_ip inet NOT NULL,
    aws_hostnames_hostname character varying
);
ALTER TABLE ONLY public.aws_events_ips_hostnames ATTACH PARTITION public.aws_events_ips_hostnames_2019_05_07to2019_05_21 FOR VALUES FROM ('2019-05-07 00:00:00') TO ('2019-05-21 00:00:00');


ALTER TABLE public.aws_events_ips_hostnames_2019_05_07to2019_05_21 OWNER TO "user";

--
-- Name: aws_events_ips_hostnames_2019_05_21to2019_06_04; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_events_ips_hostnames_2019_05_21to2019_06_04 (
    ts timestamp without time zone NOT NULL,
    is_public boolean NOT NULL,
    is_join boolean NOT NULL,
    aws_resources_id character varying NOT NULL,
    aws_ips_ip inet NOT NULL,
    aws_hostnames_hostname character varying
);
ALTER TABLE ONLY public.aws_events_ips_hostnames ATTACH PARTITION public.aws_events_ips_hostnames_2019_05_21to2019_06_04 FOR VALUES FROM ('2019-05-21 00:00:00') TO ('2019-06-04 00:00:00');


ALTER TABLE public.aws_events_ips_hostnames_2019_05_21to2019_06_04 OWNER TO "user";

--
-- Name: aws_events_ips_hostnames_2019_06_04to2019_06_18; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_events_ips_hostnames_2019_06_04to2019_06_18 (
    ts timestamp without time zone NOT NULL,
    is_public boolean NOT NULL,
    is_join boolean NOT NULL,
    aws_resources_id character varying NOT NULL,
    aws_ips_ip inet NOT NULL,
    aws_hostnames_hostname character varying
);
ALTER TABLE ONLY public.aws_events_ips_hostnames ATTACH PARTITION public.aws_events_ips_hostnames_2019_06_04to2019_06_18 FOR VALUES FROM ('2019-06-04 00:00:00') TO ('2019-06-18 00:00:00');


ALTER TABLE public.aws_events_ips_hostnames_2019_06_04to2019_06_18 OWNER TO "user";

--
-- Name: aws_events_ips_hostnames_2019_06_18to2019_07_02; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_events_ips_hostnames_2019_06_18to2019_07_02 (
    ts timestamp without time zone NOT NULL,
    is_public boolean NOT NULL,
    is_join boolean NOT NULL,
    aws_resources_id character varying NOT NULL,
    aws_ips_ip inet NOT NULL,
    aws_hostnames_hostname character varying
);
ALTER TABLE ONLY public.aws_events_ips_hostnames ATTACH PARTITION public.aws_events_ips_hostnames_2019_06_18to2019_07_02 FOR VALUES FROM ('2019-06-18 00:00:00') TO ('2019-07-02 00:00:00');


ALTER TABLE public.aws_events_ips_hostnames_2019_06_18to2019_07_02 OWNER TO "user";

--
-- Name: aws_events_ips_hostnames_2019_08_01to2019_10_30; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_events_ips_hostnames_2019_08_01to2019_10_30 (
    ts timestamp without time zone NOT NULL,
    is_public boolean NOT NULL,
    is_join boolean NOT NULL,
    aws_resources_id character varying NOT NULL,
    aws_ips_ip inet NOT NULL,
    aws_hostnames_hostname character varying
);
ALTER TABLE ONLY public.aws_events_ips_hostnames ATTACH PARTITION public.aws_events_ips_hostnames_2019_08_01to2019_10_30 FOR VALUES FROM ('2019-08-01 00:00:00') TO ('2019-10-30 00:00:00');


ALTER TABLE public.aws_events_ips_hostnames_2019_08_01to2019_10_30 OWNER TO "user";

--
-- Name: aws_hostnames; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_hostnames (
    hostname character varying NOT NULL
);


ALTER TABLE public.aws_hostnames OWNER TO "user";

--
-- Name: aws_ips; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_ips (
    ip inet NOT NULL
);


ALTER TABLE public.aws_ips OWNER TO "user";

--
-- Name: aws_private_ip_assignment; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_private_ip_assignment (
    id bigint NOT NULL,
    not_before timestamp without time zone NOT NULL,
    not_after timestamp without time zone,
    private_ip inet NOT NULL,
    aws_resource_id integer NOT NULL
);


ALTER TABLE public.aws_private_ip_assignment OWNER TO "user";

--
-- Name: aws_private_ip_assignment_id_seq; Type: SEQUENCE; Schema: public; Owner: user
--

CREATE SEQUENCE public.aws_private_ip_assignment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.aws_private_ip_assignment_id_seq OWNER TO "user";

--
-- Name: aws_private_ip_assignment_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: user
--

ALTER SEQUENCE public.aws_private_ip_assignment_id_seq OWNED BY public.aws_private_ip_assignment.id;


--
-- Name: aws_public_ip_assignment; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_public_ip_assignment (
    id bigint NOT NULL,
    not_before timestamp without time zone NOT NULL,
    not_after timestamp without time zone,
    public_ip inet NOT NULL,
    aws_hostname character varying NOT NULL,
    aws_resource_id bigint NOT NULL
);


ALTER TABLE public.aws_public_ip_assignment OWNER TO "user";

--
-- Name: aws_public_ip_assignment_id_seq; Type: SEQUENCE; Schema: public; Owner: user
--

CREATE SEQUENCE public.aws_public_ip_assignment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.aws_public_ip_assignment_id_seq OWNER TO "user";

--
-- Name: aws_public_ip_assignment_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: user
--

ALTER SEQUENCE public.aws_public_ip_assignment_id_seq OWNED BY public.aws_public_ip_assignment.id;


--
-- Name: aws_region; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_region (
    id integer NOT NULL,
    region character varying NOT NULL
);


ALTER TABLE public.aws_region OWNER TO "user";

--
-- Name: aws_region_id_seq; Type: SEQUENCE; Schema: public; Owner: user
--

CREATE SEQUENCE public.aws_region_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.aws_region_id_seq OWNER TO "user";

--
-- Name: aws_region_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: user
--

ALTER SEQUENCE public.aws_region_id_seq OWNED BY public.aws_region.id;


--
-- Name: aws_resource; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_resource (
    id bigint NOT NULL,
    arn_id character varying NOT NULL,
    aws_account_id integer NOT NULL,
    aws_region_id integer NOT NULL,
    aws_resource_type_id integer NOT NULL,
    meta jsonb
);


ALTER TABLE public.aws_resource OWNER TO "user";

--
-- Name: aws_resource_id_seq; Type: SEQUENCE; Schema: public; Owner: user
--

CREATE SEQUENCE public.aws_resource_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.aws_resource_id_seq OWNER TO "user";

--
-- Name: aws_resource_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: user
--

ALTER SEQUENCE public.aws_resource_id_seq OWNED BY public.aws_resource.id;


--
-- Name: aws_resource_type; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_resource_type (
    id integer NOT NULL,
    resource_type character varying NOT NULL
);


ALTER TABLE public.aws_resource_type OWNER TO "user";

--
-- Name: aws_resource_type_id_seq; Type: SEQUENCE; Schema: public; Owner: user
--

CREATE SEQUENCE public.aws_resource_type_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.aws_resource_type_id_seq OWNER TO "user";

--
-- Name: aws_resource_type_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: user
--

ALTER SEQUENCE public.aws_resource_type_id_seq OWNED BY public.aws_resource_type.id;


--
-- Name: aws_resources; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.aws_resources (
    id character varying NOT NULL,
    account_id character varying NOT NULL,
    region character varying NOT NULL,
    type character varying NOT NULL,
    meta jsonb
);


ALTER TABLE public.aws_resources OWNER TO "user";

--
-- Name: partitions; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.partitions (
    name character varying NOT NULL,
    created_at timestamp without time zone NOT NULL,
    partition_begin date NOT NULL,
    partition_end date NOT NULL
);


ALTER TABLE public.partitions OWNER TO "user";

--
-- Name: person; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.person (
    id integer NOT NULL,
    login character varying NOT NULL,
    email character varying NOT NULL,
    name character varying NOT NULL,
    valid boolean NOT NULL
);


ALTER TABLE public.person OWNER TO "user";

--
-- Name: person_id_seq; Type: SEQUENCE; Schema: public; Owner: user
--

CREATE SEQUENCE public.person_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.person_id_seq OWNER TO "user";

--
-- Name: person_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: user
--

ALTER SEQUENCE public.person_id_seq OWNED BY public.person.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO "user";

--
-- Name: account_champion person_id; Type: DEFAULT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.account_champion ALTER COLUMN person_id SET DEFAULT nextval('public.account_champion_person_id_seq'::regclass);


--
-- Name: account_owner person_id; Type: DEFAULT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.account_owner ALTER COLUMN person_id SET DEFAULT nextval('public.account_owner_person_id_seq'::regclass);


--
-- Name: aws_account id; Type: DEFAULT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_account ALTER COLUMN id SET DEFAULT nextval('public.aws_account_id_seq'::regclass);


--
-- Name: aws_private_ip_assignment id; Type: DEFAULT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_private_ip_assignment ALTER COLUMN id SET DEFAULT nextval('public.aws_private_ip_assignment_id_seq'::regclass);


--
-- Name: aws_public_ip_assignment id; Type: DEFAULT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_public_ip_assignment ALTER COLUMN id SET DEFAULT nextval('public.aws_public_ip_assignment_id_seq'::regclass);


--
-- Name: aws_region id; Type: DEFAULT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_region ALTER COLUMN id SET DEFAULT nextval('public.aws_region_id_seq'::regclass);


--
-- Name: aws_resource id; Type: DEFAULT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_resource ALTER COLUMN id SET DEFAULT nextval('public.aws_resource_id_seq'::regclass);


--
-- Name: aws_resource_type id; Type: DEFAULT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_resource_type ALTER COLUMN id SET DEFAULT nextval('public.aws_resource_type_id_seq'::regclass);


--
-- Name: person id; Type: DEFAULT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.person ALTER COLUMN id SET DEFAULT nextval('public.person_id_seq'::regclass);


--
-- Name: account_champion account_champion_person_id_aws_account_id_key; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.account_champion
    ADD CONSTRAINT account_champion_person_id_aws_account_id_key UNIQUE (person_id, aws_account_id);


--
-- Name: account_owner account_owner_aws_account_id_key; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.account_owner
    ADD CONSTRAINT account_owner_aws_account_id_key UNIQUE (aws_account_id);


--
-- Name: aws_account aws_account_account_key; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_account
    ADD CONSTRAINT aws_account_account_key UNIQUE (account);


--
-- Name: aws_account aws_account_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_account
    ADD CONSTRAINT aws_account_pkey PRIMARY KEY (id);


--
-- Name: aws_hostnames aws_hostnames_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_hostnames
    ADD CONSTRAINT aws_hostnames_pkey PRIMARY KEY (hostname);


--
-- Name: aws_ips aws_ips_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_ips
    ADD CONSTRAINT aws_ips_pkey PRIMARY KEY (ip);


--
-- Name: aws_private_ip_assignment aws_private_ip_assignment_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_private_ip_assignment
    ADD CONSTRAINT aws_private_ip_assignment_pkey PRIMARY KEY (id);


--
-- Name: aws_public_ip_assignment aws_public_ip_assignment_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_public_ip_assignment
    ADD CONSTRAINT aws_public_ip_assignment_pkey PRIMARY KEY (id);


--
-- Name: aws_region aws_region_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_region
    ADD CONSTRAINT aws_region_pkey PRIMARY KEY (id);


--
-- Name: aws_region aws_region_region_key; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_region
    ADD CONSTRAINT aws_region_region_key UNIQUE (region);


--
-- Name: aws_resource aws_resource_arn_id_key; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_resource
    ADD CONSTRAINT aws_resource_arn_id_key UNIQUE (arn_id);


--
-- Name: aws_resource aws_resource_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_resource
    ADD CONSTRAINT aws_resource_pkey PRIMARY KEY (id);


--
-- Name: aws_resource_type aws_resource_type_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_resource_type
    ADD CONSTRAINT aws_resource_type_pkey PRIMARY KEY (id);


--
-- Name: aws_resource_type aws_resource_type_resource_type_key; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_resource_type
    ADD CONSTRAINT aws_resource_type_resource_type_key UNIQUE (resource_type);


--
-- Name: aws_resources aws_resources_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_resources
    ADD CONSTRAINT aws_resources_pkey PRIMARY KEY (id);


--
-- Name: partitions partitions_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.partitions
    ADD CONSTRAINT partitions_pkey PRIMARY KEY (name);


--
-- Name: person person_email_key; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.person
    ADD CONSTRAINT person_email_key UNIQUE (email);


--
-- Name: person person_login_key; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.person
    ADD CONSTRAINT person_login_key UNIQUE (login);


--
-- Name: person person_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.person
    ADD CONSTRAINT person_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: aws_private_ip_assignment_idx_no_after; Type: INDEX; Schema: public; Owner: user
--

CREATE UNIQUE INDEX aws_private_ip_assignment_idx_no_after ON public.aws_private_ip_assignment USING btree (not_before, private_ip, aws_resource_id) WHERE (not_after IS NULL);


--
-- Name: aws_private_ip_assignment_idx_no_before; Type: INDEX; Schema: public; Owner: user
--

CREATE UNIQUE INDEX aws_private_ip_assignment_idx_no_before ON public.aws_private_ip_assignment USING btree (not_after, private_ip, aws_resource_id) WHERE (not_before IS NULL);


--
-- Name: aws_public_ip_assignment_idx_no_after; Type: INDEX; Schema: public; Owner: user
--

CREATE UNIQUE INDEX aws_public_ip_assignment_idx_no_after ON public.aws_public_ip_assignment USING btree (not_before, public_ip, aws_resource_id) WHERE (not_after IS NULL);


--
-- Name: aws_public_ip_assignment_idx_no_before; Type: INDEX; Schema: public; Owner: user
--

CREATE UNIQUE INDEX aws_public_ip_assignment_idx_no_before ON public.aws_public_ip_assignment USING btree (not_after, public_ip, aws_resource_id) WHERE (not_before IS NULL);


--
-- Name: idx_aws_hostname; Type: INDEX; Schema: public; Owner: user
--

CREATE INDEX idx_aws_hostname ON public.aws_public_ip_assignment USING btree (aws_hostname);


--
-- Name: idx_private_ip; Type: INDEX; Schema: public; Owner: user
--

CREATE INDEX idx_private_ip ON public.aws_private_ip_assignment USING btree (private_ip);


--
-- Name: idx_public_ip; Type: INDEX; Schema: public; Owner: user
--

CREATE INDEX idx_public_ip ON public.aws_public_ip_assignment USING btree (public_ip);


--
-- Name: account_champion account_champion_aws_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.account_champion
    ADD CONSTRAINT account_champion_aws_account_id_fkey FOREIGN KEY (aws_account_id) REFERENCES public.aws_account(id);


--
-- Name: account_champion account_champion_person_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.account_champion
    ADD CONSTRAINT account_champion_person_id_fkey FOREIGN KEY (person_id) REFERENCES public.person(id);


--
-- Name: account_owner account_owner_aws_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.account_owner
    ADD CONSTRAINT account_owner_aws_account_id_fkey FOREIGN KEY (aws_account_id) REFERENCES public.aws_account(id);


--
-- Name: account_owner account_owner_person_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.account_owner
    ADD CONSTRAINT account_owner_person_id_fkey FOREIGN KEY (person_id) REFERENCES public.person(id);


--
-- Name: aws_events_ips_hostnames aws_events_ips_hostnames_aws_hostnames_hostname_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE public.aws_events_ips_hostnames
    ADD CONSTRAINT aws_events_ips_hostnames_aws_hostnames_hostname_fkey FOREIGN KEY (aws_hostnames_hostname) REFERENCES public.aws_hostnames(hostname);


--
-- Name: aws_events_ips_hostnames aws_events_ips_hostnames_aws_ips_ip_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE public.aws_events_ips_hostnames
    ADD CONSTRAINT aws_events_ips_hostnames_aws_ips_ip_fkey FOREIGN KEY (aws_ips_ip) REFERENCES public.aws_ips(ip);


--
-- Name: aws_events_ips_hostnames aws_events_ips_hostnames_aws_resources_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE public.aws_events_ips_hostnames
    ADD CONSTRAINT aws_events_ips_hostnames_aws_resources_id_fkey FOREIGN KEY (aws_resources_id) REFERENCES public.aws_resources(id);


--
-- Name: aws_private_ip_assignment aws_private_ip_assignment_aws_resource_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_private_ip_assignment
    ADD CONSTRAINT aws_private_ip_assignment_aws_resource_id_fkey FOREIGN KEY (aws_resource_id) REFERENCES public.aws_resource(id);


--
-- Name: aws_public_ip_assignment aws_public_ip_assignment_aws_resource_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_public_ip_assignment
    ADD CONSTRAINT aws_public_ip_assignment_aws_resource_id_fkey FOREIGN KEY (aws_resource_id) REFERENCES public.aws_resource(id);


--
-- Name: aws_resource aws_resource_aws_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_resource
    ADD CONSTRAINT aws_resource_aws_account_id_fkey FOREIGN KEY (aws_account_id) REFERENCES public.aws_account(id);


--
-- Name: aws_resource aws_resource_aws_region_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_resource
    ADD CONSTRAINT aws_resource_aws_region_id_fkey FOREIGN KEY (aws_region_id) REFERENCES public.aws_region(id);


--
-- Name: aws_resource aws_resource_aws_resource_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.aws_resource
    ADD CONSTRAINT aws_resource_aws_resource_type_id_fkey FOREIGN KEY (aws_resource_type_id) REFERENCES public.aws_resource_type(id);


--
-- PostgreSQL database dump complete
--


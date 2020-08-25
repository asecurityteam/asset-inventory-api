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

SET default_tablespace = '';

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
-- PostgreSQL database dump complete
--


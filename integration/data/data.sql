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
-- Data for Name: aws_hostnames; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.aws_hostnames (hostname) VALUES ('ec2-54-0-0-1.compute-1.amazonaws.com');
INSERT INTO public.aws_hostnames (hostname) VALUES ('ec2-3-0-0-1.compute-1.amazonaws.com');
INSERT INTO public.aws_hostnames (hostname) VALUES ('ec2-3-0-0-2.compute-1.amazonaws.com');
INSERT INTO public.aws_hostnames (hostname) VALUES ('ec2-54-0-0-2.compute-1.amazonaws.com');


--
-- Data for Name: aws_ips; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.aws_ips (ip) VALUES ('172.0.0.1');
INSERT INTO public.aws_ips (ip) VALUES ('54.0.0.1');
INSERT INTO public.aws_ips (ip) VALUES ('3.0.0.1');
INSERT INTO public.aws_ips (ip) VALUES ('3.0.0.2');
INSERT INTO public.aws_ips (ip) VALUES ('54.0.0.2');
INSERT INTO public.aws_ips (ip) VALUES ('172.0.0.2');


--
-- Data for Name: aws_resources; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.aws_resources (id, account_id, region, type, meta) VALUES ('arn:aws:ec2:us-east-1:100000000001:instance/i-00000000000000001', '100000000001', 'us-east-1', 'AWS::EC2::Instance', NULL);
INSERT INTO public.aws_resources (id, account_id, region, type, meta) VALUES ('arn:aws:ec2:us-east-1:100000000001:instance/i-00000000000000002', '100000000001', 'us-east-1', 'AWS::EC2::Instance', NULL);
INSERT INTO public.aws_resources (id, account_id, region, type, meta) VALUES ('arn:aws:ec2:us-east-1:200000000002:instance/i-00000000000000001', '200000000002', 'us-east-1', 'AWS::EC2::Instance', NULL);
INSERT INTO public.aws_resources (id, account_id, region, type, meta) VALUES ('arn:aws:ec2:us-east-1:300000000003:instance/i-00000000000000001', '300000000003', 'us-east-1', 'AWS::EC2::Instance', NULL);
INSERT INTO public.aws_resources (id, account_id, region, type, meta) VALUES ('arn:aws:ec2:us-east-1:400000000004:instance/i-00000000000000001', '400000000004', 'us-east-1', 'AWS::EC2::Instance', NULL);
INSERT INTO public.aws_resources (id, account_id, region, type, meta) VALUES ('arn:aws:ec2:us-east-1:100000000001:instance/i-00000000000000003', '100000000001', 'us-east-1', 'AWS::EC2::Instance', NULL);


--
-- Data for Name: partitions; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.partitions (name, created_at, partition_begin, partition_end) VALUES ('aws_events_ips_hostnames_2019_08_01to2019_10_30', '2020-07-01 04:25:12.020496', '2019-08-01', '2019-10-30');
INSERT INTO public.partitions (name, created_at, partition_begin, partition_end) VALUES ('aws_events_ips_hostnames_2019_06_18to2019_07_02', '2020-07-01 04:25:12.025858', '2019-06-18', '2019-07-02');
INSERT INTO public.partitions (name, created_at, partition_begin, partition_end) VALUES ('aws_events_ips_hostnames_2019_06_04to2019_06_18', '2020-07-01 04:25:12.030184', '2019-06-04', '2019-06-18');
INSERT INTO public.partitions (name, created_at, partition_begin, partition_end) VALUES ('aws_events_ips_hostnames_2019_05_21to2019_06_04', '2020-07-01 04:25:12.03457', '2019-05-21', '2019-06-04');
INSERT INTO public.partitions (name, created_at, partition_begin, partition_end) VALUES ('aws_events_ips_hostnames_2019_05_07to2019_05_21', '2020-07-01 04:25:12.038916', '2019-05-07', '2019-05-21');


--
-- Data for Name: aws_events_ips_hostnames_2019_05_07to2019_05_21; Type: TABLE DATA; Schema: public; Owner: user
--



--
-- Data for Name: aws_events_ips_hostnames_2019_05_21to2019_06_04; Type: TABLE DATA; Schema: public; Owner: user
--



--
-- Data for Name: aws_events_ips_hostnames_2019_06_04to2019_06_18; Type: TABLE DATA; Schema: public; Owner: user
--



--
-- Data for Name: aws_events_ips_hostnames_2019_06_18to2019_07_02; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.aws_events_ips_hostnames_2019_06_18to2019_07_02 (ts, is_public, is_join, aws_resources_id, aws_ips_ip, aws_hostnames_hostname) VALUES ('2019-06-18 20:58:14.329', false, true, 'arn:aws:ec2:us-east-1:100000000001:instance/i-00000000000000001', '172.0.0.1', NULL);
INSERT INTO public.aws_events_ips_hostnames_2019_06_18to2019_07_02 (ts, is_public, is_join, aws_resources_id, aws_ips_ip, aws_hostnames_hostname) VALUES ('2019-06-18 20:58:45.814', true, true, 'arn:aws:ec2:us-east-1:100000000001:instance/i-00000000000000002', '54.0.0.1', 'ec2-54-0-0-1.compute-1.amazonaws.com');
INSERT INTO public.aws_events_ips_hostnames_2019_06_18to2019_07_02 (ts, is_public, is_join, aws_resources_id, aws_ips_ip, aws_hostnames_hostname) VALUES ('2019-06-18 20:59:05.857', true, false, 'arn:aws:ec2:us-east-1:200000000002:instance/i-00000000000000001', '3.0.0.1', 'ec2-3-0-0-1.compute-1.amazonaws.com');
INSERT INTO public.aws_events_ips_hostnames_2019_06_18to2019_07_02 (ts, is_public, is_join, aws_resources_id, aws_ips_ip, aws_hostnames_hostname) VALUES ('2019-06-18 20:59:13.397', true, false, 'arn:aws:ec2:us-east-1:300000000003:instance/i-00000000000000001', '3.0.0.2', 'ec2-3-0-0-2.compute-1.amazonaws.com');
INSERT INTO public.aws_events_ips_hostnames_2019_06_18to2019_07_02 (ts, is_public, is_join, aws_resources_id, aws_ips_ip, aws_hostnames_hostname) VALUES ('2019-06-18 21:13:34.022', false, false, 'arn:aws:ec2:us-east-1:400000000004:instance/i-00000000000000001', '172.0.0.2', NULL);
INSERT INTO public.aws_events_ips_hostnames_2019_06_18to2019_07_02 (ts, is_public, is_join, aws_resources_id, aws_ips_ip, aws_hostnames_hostname) VALUES ('2019-06-18 21:21:47.037', true, false, 'arn:aws:ec2:us-east-1:100000000001:instance/i-00000000000000003', '54.0.0.2', 'ec2-54-0-0-2.compute-1.amazonaws.com');


--
-- Data for Name: aws_events_ips_hostnames_2019_08_01to2019_10_30; Type: TABLE DATA; Schema: public; Owner: user
--



--
-- PostgreSQL database dump complete
--


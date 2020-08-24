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
-- Data for Name: aws_account; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.aws_account (id, account) VALUES (16, '345678901212');
INSERT INTO public.aws_account (id, account) VALUES (2, '234567890121');
INSERT INTO public.aws_account (id, account) VALUES (1, '123456789012');


--
-- Data for Name: person; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.person (id, login, email, name, valid) VALUES (11, 'login1', 'email1@atlassian.com', 'Test1', true);
INSERT INTO public.person (id, login, email, name, valid) VALUES (12, 'login2', 'email2@atlassian.com', 'Test2', true);
INSERT INTO public.person (id, login, email, name, valid) VALUES (13, 'login3', 'email3@atlassian.com', 'Test3', true);


--
-- Data for Name: account_champion; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.account_champion (person_id, aws_account_id) VALUES (11, 1);
INSERT INTO public.account_champion (person_id, aws_account_id) VALUES (12, 2);
INSERT INTO public.account_champion (person_id, aws_account_id) VALUES (13, 16);


--
-- Data for Name: account_owner; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.account_owner (person_id, aws_account_id) VALUES (11, 1);
INSERT INTO public.account_owner (person_id, aws_account_id) VALUES (12, 2);
INSERT INTO public.account_owner (person_id, aws_account_id) VALUES (13, 16);


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
-- Data for Name: aws_region; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.aws_region (id, region) VALUES (1, 'us-east-1');
INSERT INTO public.aws_region (id, region) VALUES (3, 'ap-southeast-2');
INSERT INTO public.aws_region (id, region) VALUES (7, 'ap-southeast-1');


--
-- Data for Name: aws_resource_type; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.aws_resource_type (id, resource_type) VALUES (1, 'AWS::EC2::Instance');


--
-- Data for Name: aws_resource; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.aws_resource (id, arn_id, aws_account_id, aws_region_id, aws_resource_type_id, meta) VALUES (1542460, 'i-00000000000000005', 2, 1, 1, NULL);
INSERT INTO public.aws_resource (id, arn_id, aws_account_id, aws_region_id, aws_resource_type_id, meta) VALUES (1542449, 'i-00000000000000002', 16, 3, 1, NULL);
INSERT INTO public.aws_resource (id, arn_id, aws_account_id, aws_region_id, aws_resource_type_id, meta) VALUES (1542446, 'i-00000000000000001', 16, 7, 1, NULL);
INSERT INTO public.aws_resource (id, arn_id, aws_account_id, aws_region_id, aws_resource_type_id, meta) VALUES (1542457, 'i-00000000000000004', 2, 1, 1, NULL);
INSERT INTO public.aws_resource (id, arn_id, aws_account_id, aws_region_id, aws_resource_type_id, meta) VALUES (1542453, 'i-00000000000000003', 1, 1, 1, NULL);
INSERT INTO public.aws_resource (id, arn_id, aws_account_id, aws_region_id, aws_resource_type_id, meta) VALUES (1542464, 'i-00000000000000006', 1, 3, 1, NULL);


--
-- Data for Name: aws_private_ip_assignment; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.aws_private_ip_assignment (id, not_before, not_after, private_ip, aws_resource_id) VALUES (652967, '2019-07-25 07:50:30.174', NULL, '10.0.0.4', 1542460);
INSERT INTO public.aws_private_ip_assignment (id, not_before, not_after, private_ip, aws_resource_id) VALUES (652964, '2019-07-25 07:50:30.174', NULL, '10.0.0.1', 1542457);
INSERT INTO public.aws_private_ip_assignment (id, not_before, not_after, private_ip, aws_resource_id) VALUES (652965, '2019-07-25 07:50:30.174', NULL, '10.0.0.2', 1542457);
INSERT INTO public.aws_private_ip_assignment (id, not_before, not_after, private_ip, aws_resource_id) VALUES (652963, '2019-07-25 07:50:29.264', '2019-07-25 08:09:54.306', '10.0.0.5', 1542453);
INSERT INTO public.aws_private_ip_assignment (id, not_before, not_after, private_ip, aws_resource_id) VALUES (652962, '2019-07-25 07:50:27.538', '2019-07-25 08:54:41.318', '10.0.0.6', 1542449);
INSERT INTO public.aws_private_ip_assignment (id, not_before, not_after, private_ip, aws_resource_id) VALUES (652959, '2019-07-25 07:50:27.356', '2019-07-25 08:55:02.726', '10.0.0.9', 1542446);
INSERT INTO public.aws_private_ip_assignment (id, not_before, not_after, private_ip, aws_resource_id) VALUES (652961, '2019-07-25 07:50:27.538', '2019-07-25 08:54:41.318', '10.0.0.7', 1542449);
INSERT INTO public.aws_private_ip_assignment (id, not_before, not_after, private_ip, aws_resource_id) VALUES (652966, '2019-07-25 07:50:30.174', NULL, '10.0.0.3', 1542460);
INSERT INTO public.aws_private_ip_assignment (id, not_before, not_after, private_ip, aws_resource_id) VALUES (652960, '2019-07-25 07:50:27.356', '2019-07-25 08:55:02.726', '10.0.0.8', 1542446);


--
-- Data for Name: aws_public_ip_assignment; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.aws_public_ip_assignment (id, not_before, not_after, public_ip, aws_hostname, aws_resource_id) VALUES (213405, '2019-07-25 07:50:30.174', NULL, '3.0.0.1', 'ec2-3-0-0-1.compute-1.amazonaws.com', 1542457);
INSERT INTO public.aws_public_ip_assignment (id, not_before, not_after, public_ip, aws_hostname, aws_resource_id) VALUES (213403, '2019-07-25 07:50:27.356', '2019-07-25 08:55:02.726', '3.0.0.4', 'ec2-3-0-0-4.ap-southeast-1.compute.amazonaws.com', 1542446);
INSERT INTO public.aws_public_ip_assignment (id, not_before, not_after, public_ip, aws_hostname, aws_resource_id) VALUES (213404, '2019-07-25 07:50:27.538', '2019-07-25 08:54:41.318', '3.0.0.3', 'ec2-3-0-0-3.ap-southeast-2.compute.amazonaws.com', 1542449);
INSERT INTO public.aws_public_ip_assignment (id, not_before, not_after, public_ip, aws_hostname, aws_resource_id) VALUES (213406, '2019-07-25 07:50:30.174', NULL, '3.0.0.2', 'ec2-3-0-0-2.compute-1.amazonaws.com', 1542460);
INSERT INTO public.aws_public_ip_assignment (id, not_before, not_after, public_ip, aws_hostname, aws_resource_id) VALUES (213407, '2020-07-20 07:50:30.174', NULL, '127.0.0.1', 'ec2-3-0-0-5.compute-1.amazonaws.com', 1542464);


--
-- Data for Name: partitions; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.partitions (name, created_at, partition_begin, partition_end) VALUES ('aws_events_ips_hostnames_2019_08_01to2019_10_30', '2020-07-01 04:25:12.020496', '2019-08-01', '2019-10-30');
INSERT INTO public.partitions (name, created_at, partition_begin, partition_end) VALUES ('aws_events_ips_hostnames_2019_06_18to2019_07_02', '2020-07-01 04:25:12.025858', '2019-06-18', '2019-07-02');
INSERT INTO public.partitions (name, created_at, partition_begin, partition_end) VALUES ('aws_events_ips_hostnames_2019_06_04to2019_06_18', '2020-07-01 04:25:12.030184', '2019-06-04', '2019-06-18');
INSERT INTO public.partitions (name, created_at, partition_begin, partition_end) VALUES ('aws_events_ips_hostnames_2019_05_21to2019_06_04', '2020-07-01 04:25:12.03457', '2019-05-21', '2019-06-04');
INSERT INTO public.partitions (name, created_at, partition_begin, partition_end) VALUES ('aws_events_ips_hostnames_2019_05_07to2019_05_21', '2020-07-01 04:25:12.038916', '2019-05-07', '2019-05-21');


--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: user
--

INSERT INTO public.schema_migrations (version, dirty) VALUES (6, false);


--
-- Name: account_champion_person_id_seq; Type: SEQUENCE SET; Schema: public; Owner: user
--

SELECT pg_catalog.setval('public.account_champion_person_id_seq', 1, false);


--
-- Name: account_owner_person_id_seq; Type: SEQUENCE SET; Schema: public; Owner: user
--

SELECT pg_catalog.setval('public.account_owner_person_id_seq', 1, false);


--
-- Name: aws_account_id_seq; Type: SEQUENCE SET; Schema: public; Owner: user
--

SELECT pg_catalog.setval('public.aws_account_id_seq', 1, false);


--
-- Name: aws_private_ip_assignment_id_seq; Type: SEQUENCE SET; Schema: public; Owner: user
--

SELECT pg_catalog.setval('public.aws_private_ip_assignment_id_seq', 1, false);


--
-- Name: aws_public_ip_assignment_id_seq; Type: SEQUENCE SET; Schema: public; Owner: user
--

SELECT pg_catalog.setval('public.aws_public_ip_assignment_id_seq', 1, false);


--
-- Name: aws_region_id_seq; Type: SEQUENCE SET; Schema: public; Owner: user
--

SELECT pg_catalog.setval('public.aws_region_id_seq', 1, false);


--
-- Name: aws_resource_id_seq; Type: SEQUENCE SET; Schema: public; Owner: user
--

SELECT pg_catalog.setval('public.aws_resource_id_seq', 1, false);


--
-- Name: aws_resource_type_id_seq; Type: SEQUENCE SET; Schema: public; Owner: user
--

SELECT pg_catalog.setval('public.aws_resource_type_id_seq', 1, false);


--
-- Name: person_id_seq; Type: SEQUENCE SET; Schema: public; Owner: user
--

SELECT pg_catalog.setval('public.person_id_seq', 1, false);


--
-- PostgreSQL database dump complete
--


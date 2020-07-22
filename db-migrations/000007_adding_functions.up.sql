-- schema changes to create functions for APIs
BEGIN;

CREATE OR REPLACE FUNCTION get_resource_by_arn_id(aid VARCHAR, ts TIMESTAMP)
    RETURNS TABLE
            (
                private_ip     INET,
                public_ip      INET,
                aws_hostname   VARCHAR,
                resource_type  VARCHAR,
                account        VARCHAR,
                region         VARCHAR,
                meta           JSONB,
                aws_account_id INTEGER,
                t_account      VARCHAR,
                t_login        VARCHAR,
                t_email        VARCHAR,
                t_name         VARCHAR,
                t_valid        BOOL,
                p_login        VARCHAR,
                p_email        VARCHAR,
                p_name         VARCHAR,
                p_valid        BOOL
            )
AS
$$
DECLARE
    var_aws_account_id INTEGER;
BEGIN
    CREATE TEMP TABLE IF NOT EXISTS temp_aws_resource_table AS
    SELECT pria.private_ip,
           puia.public_ip,
           puia.aws_hostname,
           rt.resource_type,
           aa.account,
           ar.region,
           res.meta,
           res.aws_account_id
    FROM aws_resource res
             LEFT JOIN aws_region ar ON res.aws_region_id = ar.id
             LEFT JOIN aws_account aa ON res.aws_account_id = aa.id
             LEFT JOIN aws_resource_type rt ON res.aws_resource_type_id = rt.id
             LEFT JOIN aws_public_ip_assignment puia ON res.id = puia.aws_resource_id
             LEFT JOIN aws_private_ip_assignment pria ON res.id = pria.aws_resource_id
    WHERE res.arn_id = aid
      AND puia.not_before < ts
      AND (puia.not_after IS NULL OR puia.not_after > ts)
      AND pria.not_before < ts
      AND (pria.not_after IS NULL OR pria.not_after > ts);

    SELECT temp_aws_resource_table.aws_account_id
    INTO var_aws_account_id
    FROM temp_aws_resource_table FETCH FIRST ROW ONLY;

    RETURN QUERY SELECT a.*, b.*
                 FROM temp_aws_resource_table AS a
                          LEFT JOIN (
                     SELECT *
                     FROM get_owner_and_champions_by_account_id(var_aws_account_id)
                 ) AS b ON a.account = b.t_account;
END;
$$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION get_owner_and_champions_by_account_id(id INTEGER)
    RETURNS TABLE
            (
                t_account VARCHAR,
                t_login   VARCHAR,
                t_email   VARCHAR,
                t_name    VARCHAR,
                t_valid   BOOL,
                p_login   VARCHAR,
                p_email   VARCHAR,
                p_name    VARCHAR,
                p_valid   BOOL
            )
AS
$$
BEGIN
    CREATE TEMP TABLE IF NOT EXISTS temp_account_owner_table AS
    SELECT aa.account,
           ow.login,
           ow.email,
           ow.name,
           ow.valid,
           ac.person_id
    FROM account_owner ao
             LEFT JOIN aws_account aa ON ao.aws_account_id = aa.id
             LEFT JOIN person ow ON ao.person_id = ow.id
             LEFT JOIN account_champion ac ON ao.aws_account_id = ac.aws_account_id
    WHERE ao.aws_account_id = get_owner_and_champions_by_account_id.id;

    RETURN QUERY SELECT t.account,
                        t.login,
                        t.email,
                        t.name,
                        t.valid,
                        p.login,
                        p.email,
                        p.name,
                        p.valid
                 FROM temp_account_owner_table t
                          LEFT JOIN person p ON t.person_id = p.id;
END;
$$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION get_resource_by_hostname(name VARCHAR, ts TIMESTAMP)
    RETURNS TABLE
            (
                public_ip     INET,
                aws_hostname  VARCHAR,
                arn_id        VARCHAR,
                meta          JSONB,
                region        VARCHAR,
                resource_type VARCHAR,
                account       VARCHAR,
                id            INTEGER,
                t_account     VARCHAR,
                t_login       VARCHAR,
                t_email       VARCHAR,
                t_name        VARCHAR,
                t_valid       BOOL,
                p_login       VARCHAR,
                p_email       VARCHAR,
                p_name        VARCHAR,
                p_valid       BOOL
            )
AS
$$
DECLARE
    var_account_id INTEGER;
BEGIN
    CREATE TEMP TABLE IF NOT EXISTS temp_hostname_table AS
    SELECT ia.public_ip,
           ia.aws_hostname,
           res.arn_id,
           res.meta,
           ar.region,
           rt.resource_type,
           aa.account,
           aa.id
    FROM aws_public_ip_assignment ia
             LEFT JOIN aws_resource res ON ia.aws_resource_id = res.id
             LEFT JOIN aws_region ar ON res.aws_region_id = ar.id
             LEFT JOIN aws_resource_type rt ON res.aws_resource_type_id = rt.id
             LEFT JOIN aws_account aa ON res.aws_account_id = aa.id
    WHERE ia.aws_hostname = name
      AND ia.not_before < ts
      AND (ia.not_after IS NULL OR ia.not_after > ts);

    SELECT temp_hostname_table.id INTO var_account_id FROM temp_hostname_table FETCH FIRST ROW ONLY;

    RETURN QUERY SELECT a.*, b.*
                 FROM temp_hostname_table AS a
                          LEFT JOIN (
                     SELECT *
                     FROM get_owner_and_champions_by_account_id(var_account_id)
                 ) AS b ON a.account = b.t_account;
END;
$$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION get_resource_by_private_ip(pip INET, ts TIMESTAMP)
    RETURNS TABLE
            (
                private_ip    INET,
                arn_id        VARCHAR,
                meta          JSONB,
                region        VARCHAR,
                resource_type VARCHAR,
                account       VARCHAR,
                id            INTEGER,
                t_account     VARCHAR,
                t_login       VARCHAR,
                t_email       VARCHAR,
                t_name        VARCHAR,
                t_valid       BOOL,
                p_login       VARCHAR,
                p_email       VARCHAR,
                p_name        VARCHAR,
                p_valid       BOOL
            )
AS
$$
DECLARE
    var_account_id INTEGER;
BEGIN
    CREATE TEMP TABLE IF NOT EXISTS temp_private_ip_table AS
    SELECT ia.private_ip,
           res.arn_id,
           res.meta,
           ar.region,
           rt.resource_type,
           aa.account,
           aa.id
    FROM aws_private_ip_assignment ia
             LEFT JOIN aws_resource res ON ia.aws_resource_id = res.id
             LEFT JOIN aws_region ar ON res.aws_region_id = ar.id
             LEFT JOIN aws_resource_type rt ON res.aws_resource_type_id = rt.id
             LEFT JOIN aws_account aa ON res.aws_account_id = aa.id
    WHERE ia.private_ip = pip
      AND ia.not_before < ts
      AND (ia.not_after IS NULL OR ia.not_after > ts);

    SELECT temp_private_ip_table.id INTO var_account_id FROM temp_private_ip_table FETCH FIRST ROW ONLY;

    RETURN QUERY SELECT a.*, b.*
                 FROM temp_private_ip_table AS a
                          LEFT JOIN (
                     SELECT *
                     FROM get_owner_and_champions_by_account_id(var_account_id)
                 ) AS b ON a.account = b.t_account;
END;
$$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION get_resource_by_public_ip(pip INET, ts TIMESTAMP)
    RETURNS TABLE
            (
                public_ip     INET,
                aws_hostname  VARCHAR,
                arn_id        VARCHAR,
                meta          JSONB,
                region        VARCHAR,
                resource_type VARCHAR,
                account       VARCHAR,
                id            INTEGER,
                t_account     VARCHAR,
                t_login       VARCHAR,
                t_email       VARCHAR,
                t_name        VARCHAR,
                t_valid       BOOL,
                p_login       VARCHAR,
                p_email       VARCHAR,
                p_name        VARCHAR,
                p_valid       BOOL
            )
AS
$$
DECLARE
    var_account_id INTEGER;
BEGIN
    CREATE TEMP TABLE IF NOT EXISTS temp_public_ip_table AS
    SELECT ia.public_ip,
           ia.aws_hostname,
           res.arn_id,
           res.meta,
           ar.region,
           rt.resource_type,
           aa.account,
           aa.id
    FROM aws_public_ip_assignment ia
             LEFT JOIN aws_resource res ON ia.aws_resource_id = res.id
             LEFT JOIN aws_region ar ON res.aws_region_id = ar.id
             LEFT JOIN aws_resource_type rt ON res.aws_resource_type_id = rt.id
             LEFT JOIN aws_account aa ON res.aws_account_id = aa.id
    WHERE ia.public_ip = pip
      AND ia.not_before < ts
      AND (ia.not_after IS NULL OR ia.not_after > ts);

    SELECT temp_public_ip_table.id INTO var_account_id FROM temp_public_ip_table FETCH FIRST ROW ONLY;

    RETURN QUERY SELECT a.*, b.*
                 FROM temp_public_ip_table AS a
                          LEFT JOIN (
                     SELECT *
                     FROM get_owner_and_champions_by_account_id(var_account_id)
                 ) AS b ON a.account = b.t_account;
END;
$$
    LANGUAGE 'plpgsql';

COMMIT;

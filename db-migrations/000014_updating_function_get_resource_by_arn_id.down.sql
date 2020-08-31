-- Reverting changes made to function get_resource_by_arn_id
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
BEGIN
    RETURN QUERY WITH wres AS (SELECT pria.private_ip,
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
                                 AND (pria.not_after IS NULL OR pria.not_after > ts))
                 SELECT wres.private_ip,
                        wres.public_ip,
                        wres.aws_hostname,
                        wres.resource_type,
                        wres.account,
                        wres.region,
                        wres.meta,
                        wres.aws_account_id,
                        b.t_account,
                        b.t_login,
                        b.t_email,
                        b.t_name,
                        b.t_valid,
                        b.p_login,
                        b.p_email,
                        b.p_name,
                        b.p_valid
                 FROM wres
                          LEFT JOIN
                      (
                          SELECT distinct iwres.aws_account_id, f.*
                          FROM wres iwres,
                               LATERAL get_owner_and_champions_by_account_id(iwres.aws_account_id) f
                      ) b
                      ON wres.account = b.t_account;
END;
$$
    LANGUAGE 'plpgsql';

COMMIT;

-- WARNING!!! it is not possible to roll back w/o back-fill of data from new schema or some other source
DO
$$BEGIN
    RAISE
        EXCEPTION
        'This migration can not be rolled back with data recovery in legacy schema. Consider recovering from backup or back-filling.';
END$$
;

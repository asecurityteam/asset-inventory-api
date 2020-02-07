package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"

	"github.com/pkg/errors"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
)

const (
	defaultPartitionInterval   = 90 // days
	tablePartitions            = "partitions"
	tableAWSResources          = "aws_resources"
	tableAWSIPS                = "aws_ips"
	tableAWSHostnames          = "aws_hostnames"
	tableAWSEventsIPSHostnames = "aws_events_ips_hostnames"

	added = "ADDED" // one of the network event types we track

)

type migrationDirection int

const (
	// used internally to designate migration direction
	up migrationDirection = iota
	down
)

const (
	// EmptySchemaVersion Version of database schema that cleans the database completely. Use cautiously!
	EmptySchemaVersion uint = 0
	// MinimumSchemaVersion Lowest version of database schema current code is able to handle
	MinimumSchemaVersion uint = 1
	// DualWriteSchemaVersion Lowest version of database schema that supports dual-writes
	DualWriteSchemaVersion uint = 2
)

// can't use Sprintf in a const, so...
// %s should be `aws_hostnames_hostname` or `aws_ips_ip`
const latestStatusQuery = "WITH latest_candidates AS ( " +
	"    SELECT " +
	"        *, " +
	"        MAX(ts) OVER (PARTITION BY aws_events_ips_hostnames.aws_resources_id) as max_ts " +
	"    FROM aws_events_ips_hostnames " +
	"    WHERE " +
	"        aws_events_ips_hostnames.%s = $1 AND " +
	"        aws_events_ips_hostnames.ts <= $2 " +
	"), " +
	"latest AS ( " +
	"    SELECT * " +
	"    FROM latest_candidates " +
	"    WHERE " +
	"        latest_candidates.ts = latest_candidates.max_ts AND " +
	"        latest_candidates.is_join = 'true' " +
	") " +
	"SELECT " +
	"    latest.aws_resources_id, " +
	"    latest.aws_ips_ip, " +
	"    latest.aws_hostnames_hostname, " +
	"    latest.is_public, " +
	"    latest.is_join, " +
	"    latest.ts, " +
	"    aws_resources.account_id, " +
	"    aws_resources.region, " +
	"    aws_resources.type, " +
	"    aws_resources.meta " +
	"FROM latest " +
	"    LEFT OUTER JOIN " +
	"    aws_resources ON " +
	"        latest.aws_resources_id = aws_resources.id;"

// Query to find resource by private IP using v2 schema
// nolint
const resourceByPrivateIPQuery = "SELECT ia.private_ip, " +
	"			res.arn_id, " +
	"			res.meta, " +
	"			ar.region, " +
	"			rt.resource_type, " +
	"			aa.account " +
	"	FROM aws_private_ip_assignment ia " +
	"		LEFT JOIN aws_resource res ON ia.aws_resource_id = res.id " +
	"		LEFT JOIN aws_region ar ON res.aws_account_id = ar.id " +
	"		LEFT JOIN aws_resource_type rt ON res.aws_resource_type_id = rt.id " +
	"		LEFT JOIN aws_account aa ON res.aws_account_id = aa.id " +
	"	WHERE ia.private_ip = $1 " +
	"		AND ia.not_before < $2 " +
	"		AND (ia.not_after IS NULL OR ia.not_after > $2);"

// Query to find resource by public IP using v2 schema
// nolint
const resourceByPublicIPQuery = "SELECT " +
	"			ia.public_ip, " +
	"			ia.hostname, " +
	"			res.arn_id, " +
	"			res.meta, " +
	"			ar.region, " +
	"			rt.resource_type, " +
	"			aa.account " +
	"	FROM aws_public_ip_assignment ia " +
	"		LEFT JOIN aws_resource res ON ia.aws_resource_id = res.id " +
	"		LEFT JOIN aws_region ar ON res.aws_account_id = ar.id " +
	"		LEFT JOIN aws_resource_type rt ON res.aws_resource_type_id = rt.id " +
	"		LEFT JOIN aws_account aa ON res.aws_account_id = aa.id " +
	"	WHERE ia.public_ip = $1 " +
	"		AND ia.not_before < $2 " +
	"		AND (ia.not_after is null or ia.not_after > $2);"

// Query to find resource by hostname using v2 schema
// nolint
const resourceByHostnameQuery = "SELECT ia.aws_hostname, " +
	"       res.arn_id, " +
	"       res.meta, " +
	"       ar.region, " +
	"       rt.resource_type, " +
	"       aa.account " +
	"	FROM aws_public_ip_assignment ia " +
	"         LEFT JOIN aws_resource res ON ia.aws_resource_id = res.id " +
	"         LEFT JOIN aws_region ar ON res.aws_account_id = ar.id " +
	"         LEFT JOIN aws_resource_type rt ON res.aws_resource_type_id = rt.id " +
	"         LEFT JOIN aws_account aa ON res.aws_account_id = aa.id " +
	"	WHERE ia.aws_hostname = $1 " +
	"  		AND ia.not_before < $2 " +
	"  		AND (ia.not_after is null OR ia.not_after > $2);"

// This query is used to retrieve all the 'active' resources (i.e. those with assigned IP/Hostname) for specific date
const bulkResourcesQuery = `
WITH lc AS (
	SELECT
	 ev.aws_resources_id, ev.aws_ips_ip, ev.aws_hostnames_hostname, ev.is_public, ev.ts , ev.is_join,
	 MAX(ev.ts) OVER (PARTITION BY ev.aws_resources_id) as max_ts
	FROM
	 aws_events_ips_hostnames as ev
	WHERE
	 ev.ts <= $1
)
SELECT
 lc.aws_resources_id, lc.aws_ips_ip, lc.aws_hostnames_hostname, lc.is_public, lc.is_join, lc.ts,
 res.account_id, res.region, res.type, res.meta
FROM
 lc
LEFT OUTER JOIN
 aws_resources as res
ON
 lc.aws_resources_id = res.id
WHERE
 lc.ts = lc.max_ts AND lc.is_join = 'true' AND res.type = $2
ORDER BY lc.ts DESC
LIMIT $3
OFFSET $4
`

//TODO Optimized query to retrieve all the 'active' resources utilizing v2 schema. Out of scope currently.

// DB represents a convenient database abstraction layer
type DB struct {
	sqldb               *sql.DB                // this is a unit test seam
	migrator            domain.StorageMigrator // another unit test seam
	once                sync.Once
	now                 func() time.Time // unit test seam
	defaultPartitionTTL int
}

// MigrateSchemaUp performs a database schema migration one version up
func (db *DB) MigrateSchemaUp(ctx context.Context) (uint, error) {
	return db.migrateSchema(ctx, up)
}

// MigrateSchemaDown performs a database schema rollback one version down
func (db *DB) MigrateSchemaDown(ctx context.Context) (uint, error) {
	return db.migrateSchema(ctx, down)
}

// MigrateSchemaToVersion performs one or more database migrations to bring schema to the specified version
func (db *DB) MigrateSchemaToVersion(ctx context.Context, version uint) error {
	return db.migrator.Migrate(version)
}

// GetSchemaVersion retrieves the current version of database schema
func (db *DB) GetSchemaVersion(ctx context.Context) (uint, error) {
	v, _, err := db.migrator.Version()
	if err == migrate.ErrNilVersion {
		// special handling for the version not being present
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (db *DB) migrateSchema(ctx context.Context, d migrationDirection) (uint, error) {
	var err error
	switch d {
	case up:
		err = db.migrator.Steps(1)
	case down:
		err = db.migrator.Steps(-1)
	default:
		return 0, errors.New("Unknown migration direction")
	}
	if err != nil {
		return 0, err
	}
	version, err := db.GetSchemaVersion(ctx)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// Init initializes a connection to a Postgres database according to the environment variables POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DATABASE
func (db *DB) Init(ctx context.Context, host string, port uint16, user string, password string, dbname string, partitionTTL int) error {
	var initErr error
	db.once.Do(func() {

		db.defaultPartitionTTL = partitionTTL

		if db.now == nil {
			db.now = time.Now
		}

		if db.sqldb == nil {
			sslmode := "disable"
			if host != "localhost" && host != "postgres" {
				sslmode = "require"
			}
			// we establish a connection against a known-to-exist dbname so we can check
			// if we need to create our desired dbname
			psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
				"password=%s dbname=%s sslmode=%s",
				host, port, user, password, "postgres", sslmode)
			pgdb, err := sql.Open("postgres", psqlInfo)
			if err != nil {
				initErr = err
				return // from the unnamed once.Do function
			}

			db.sqldb = pgdb

			psqlInfo = fmt.Sprintf("host=%s port=%d user=%s "+
				"password=%s dbname=%s sslmode=%s",
				host, port, user, password, dbname, sslmode)
			err = db.use(psqlInfo)
			if err != nil {
				initErr = err
				return // from the unnamed once.Do function
			}

			err = db.ping()
			if err != nil {
				initErr = err
				return // from the unnamed once.Do function
			}

		}
	})
	return initErr
}

// use function's intent is to close the existing connection (pgdb parameter) and open
// a new one against the desired psqlInfo connection string
func (db *DB) use(psqlInfo string) error {

	err := db.sqldb.Close()
	if err != nil {
		return err
	}
	pgdb, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}

	db.sqldb = pgdb

	return nil
}

// ping is required for Postgres connection to be fully established
func (db *DB) ping() error {

	err := db.sqldb.Ping()
	if err != nil {
		return err
	}

	return nil
}

// GeneratePartition finds the latest partition, and generate the next partition based on the previous partition's time range
func (db *DB) GeneratePartition(ctx context.Context, begin time.Time, days int) error {

	tx, err := db.sqldb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, fmt.Sprintf("LOCK TABLE %s", tablePartitions)); err != nil { // nolint
		return handleRollback(tx, err)
	}

	// If no begin time was provided, auto-create the next partition
	if begin.IsZero() {
		var latestBegin time.Time
		var latestEnd time.Time
		stmt := "SELECT partition_begin, partition_end FROM partitions ORDER BY partition_end DESC LIMIT 1"
		row := tx.QueryRowContext(ctx, stmt)
		err = row.Scan(&latestBegin, &latestEnd)
		switch err {
		case nil:
			begin = latestEnd
		case sql.ErrNoRows:
			begin = db.now()
		default:
			return handleRollback(tx, err)
		}

		daysUntilPartitionNeeded := latestEnd.Sub(db.now()).Hours() / 24
		// Check if a table has already been created in preparation for the future, or
		// check to see if it's time to create a new partition. This is a somewhat arbitrary
		// choice to auto-create the next partition 3 days in advance of its actual need, so that
		// we have time to fix any problem that may come up
		if latestBegin.After(db.now()) || daysUntilPartitionNeeded > 3 {
			return tx.Commit()
		}
	}

	if days < 1 {
		days = defaultPartitionInterval
	}

	end := begin.AddDate(0, 0, days)
	suffixTpl := `%d_%02d_%02dto%d_%02d_%02d` // YYYY_MM_DDtoYYYY_MM_DD
	partitionTableNameSuffix := fmt.Sprintf(suffixTpl, begin.Year(), begin.Month(), begin.Day(), end.Year(), end.Month(), end.Day())
	name := fmt.Sprintf(`%s_%s`, tableAWSEventsIPSHostnames, partitionTableNameSuffix)

	// nolint
	stmt := fmt.Sprintf(`INSERT INTO %s(name, created_at, partition_begin, partition_end)
    	SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
    		SELECT 1 FROM %s WHERE (
        		(partition_begin <= $3 AND partition_end > $3) OR
        		(partition_begin < $4 AND partition_end >= $4)
    		)
		)`, tablePartitions, tablePartitions)
	res, err := tx.ExecContext(ctx, stmt, name, db.now(), begin, end)
	if err != nil {
		return handleRollback(tx, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return handleRollback(tx, err)
	}
	if rows == 0 { // no rows were updated due to a conflict in our condition
		_ = tx.Rollback() // best effort rollback to close the TX
		return domain.PartitionConflict{Name: name}
	}
	stmt = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR VALUES FROM ('%s') TO ('%s')", name, tableAWSEventsIPSHostnames, begin.Format(time.RFC3339), end.Format(time.RFC3339)) // nolint
	_, err = tx.ExecContext(ctx, stmt)
	if err != nil {
		return handleRollback(tx, err)
	}
	return tx.Commit()
}

func handleRollback(tx *sql.Tx, err error) error {
	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		return fmt.Errorf("rollback error: %v while recovering from %v", rollbackErr, err)
	}
	return err
}

// GetPartitions fetches the created partitions and gets each record count in the database
func (db *DB) GetPartitions(ctx context.Context) ([]domain.Partition, error) {
	stmt := `SELECT name, created_at, partition_begin, partition_end
				FROM partitions ORDER BY partition_end DESC`
	rows, err := db.sqldb.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	partitions := make([]domain.Partition, 0)
	for rows.Next() {
		var name string
		var createdAt time.Time
		var begin time.Time
		var end time.Time
		if err := rows.Scan(&name, &createdAt, &begin, &end); err != nil {
			_ = rows.Close()
			return nil, err
		}
		stmt2 := fmt.Sprintf("SELECT count(*) FROM %s", name) //nolint
		row := db.sqldb.QueryRowContext(ctx, stmt2)
		var count int
		if err := row.Scan(&count); err != nil {
			return nil, err
		}
		partitions = append(partitions, domain.Partition{
			Name:      name,
			CreatedAt: createdAt,
			Begin:     begin,
			End:       end,
			Count:     count,
		})
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return partitions, nil
}

// DeletePartitions deletes partitions by name.
func (db *DB) DeletePartitions(ctx context.Context, name string) error {
	tx, err := db.sqldb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, fmt.Sprintf("LOCK TABLE %s", tablePartitions)); err != nil { // nolint
		return handleRollback(tx, err)
	}

	stmt := "DELETE FROM partitions WHERE name = $1"
	res, err := tx.ExecContext(ctx, stmt, name)
	if err != nil {
		return handleRollback(tx, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return handleRollback(tx, err)
	}
	if rows == 0 { //no rows deleted due to named partition not found
		err := domain.NotFoundPartition{Name: name}
		return handleRollback(tx, err)
	}

	stmt = fmt.Sprintf("DROP TABLE %s", name)
	_, err = tx.ExecContext(ctx, stmt)
	if err != nil {
		return handleRollback(tx, err)
	}
	return tx.Commit()
}

// Store an implementation of the Storage interface that records to a database
func (db *DB) Store(ctx context.Context, cloudAssetChanges domain.CloudAssetChanges) error {

	tx, err := db.sqldb.Begin()
	if err != nil {
		return err
	}

	if err = db.saveResource(ctx, cloudAssetChanges, tx); err == nil {
		for _, val := range cloudAssetChanges.Changes {
			err = db.recordNetworkChanges(ctx, cloudAssetChanges.ARN, cloudAssetChanges.ChangeTime, val, tx)
			if err != nil {
				break
			}
		}
	}

	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Wrap(rollbackErr, err.Error()) // so we don't lose the original error
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	ver, err := db.GetSchemaVersion(ctx)
	if err != nil || ver < DualWriteSchemaVersion {
		return nil
	}
	return db.StoreV2(ctx, cloudAssetChanges)
}

// StoreV2 Currently public to allow testing separately.Storage interface implementation that records to a database using new schema, to replace Store in the future.
func (db *DB) StoreV2(ctx context.Context, cloudAssetChanges domain.CloudAssetChanges) error {
	tx, err := db.sqldb.Begin()
	if err != nil {
		return err
	}
	if err = db.ensureResourceExists(ctx, cloudAssetChanges, tx); err == nil {
		arnID := resIDFromARN(cloudAssetChanges.ARN)
		for _, val := range cloudAssetChanges.Changes {
			for _, ip := range val.PrivateIPAddresses {
				if strings.EqualFold(added, val.ChangeType) {
					err = db.assignPrivateIP(ctx, tx, arnID, ip, cloudAssetChanges.ChangeTime)
				} else {
					err = db.releasePrivateIP(ctx, tx, arnID, ip, cloudAssetChanges.ChangeTime)
				}
				if err != nil {
					break
				}
			}
			for ix, ip := range val.PublicIPAddresses {
				hostname := val.Hostnames[ix]
				if strings.EqualFold(added, val.ChangeType) {
					err = db.assignPublicIP(ctx, tx, arnID, ip, hostname, cloudAssetChanges.ChangeTime)
				} else {
					err = db.releasePublicIP(ctx, tx, arnID, ip, hostname, cloudAssetChanges.ChangeTime)
				}
			}
			if err != nil {
				break
			}
		}
	}
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Wrap(rollbackErr, err.Error()) // so we don't lose the original error
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (db *DB) ensureResourceExists(ctx context.Context, cloudAssetChanges domain.CloudAssetChanges, tx *sql.Tx) error {
	// reading pre-requisite to understand this query - https://www.postgresql.org/docs/current/queries-with.html
	const createResourceQuery string = `
with sel as (
    select val.arn_id,
           val.region,
           aws_region.id        as aws_region_id,
           val.account,
           aws_account.id       as aws_account_id,
           val.resource_type,
           aws_resource_type.id as aws_resource_type_id,
           val.meta
    from (
             values (text $1, text $2, text $3, text $4, jsonb $5)
         ) val (arn_id, region, account, resource_type, meta)
             left join aws_region using (region)
             left join aws_account using (account)
             left join aws_resource_type using (resource_type)),
     ins_aws_region as (
         insert into aws_region (region)
             select distinct region from sel where aws_region_id is null
             returning id as aws_region_id, region
     ),
     ins_aws_account as (
         insert into aws_account (account)
             select distinct account from sel where aws_account_id is null
             returning id as aws_account_id, account
     ),
     ins_aws_resource_type as (
         insert into aws_resource_type (resource_type)
             select distinct resource_type from sel where aws_resource_type_id is null
             returning id as aws_resource_type_id, resource_type
     )
insert
into aws_resource (arn_id, aws_region_id, aws_account_id, aws_resource_type_id, meta)
select sel.arn_id,
       coalesce(sel.aws_region_id, ins_aws_region.aws_region_id),
       coalesce(sel.aws_account_id, ins_aws_account.aws_account_id),
       coalesce(sel.aws_resource_type_id, ins_aws_resource_type.aws_resource_type_id),
       sel.meta
from sel
         left join ins_aws_region using (region)
         left join ins_aws_account using (account)
         left join ins_aws_resource_type using (resource_type)
on conflict do nothing
`
	tagsBytes, _ := json.Marshal(cloudAssetChanges.Tags) // an error here is not possible considering json.Marshal is taking a simple map or nil
	if _, err := tx.ExecContext(ctx,
		createResourceQuery,
		resIDFromARN(cloudAssetChanges.ARN),
		cloudAssetChanges.Region,
		cloudAssetChanges.AccountID,
		cloudAssetChanges.ResourceType,
		tagsBytes); err != nil {
		return err
	}
	return nil
}

// extract unique resource ID from full ARN format
// it is always the part after the last /
// https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
func resIDFromARN(ARN string) string {
	parts := strings.SplitAfterN(ARN, "/", -1)
	return parts[len(parts)-1]
}

func (db *DB) saveResource(ctx context.Context, cloudAssetChanges domain.CloudAssetChanges, tx *sql.Tx) error {
	// You won't get an ID back if nothing was done.  Also, this lib won't return the ID anyway even without the "ON CONFLICT DO NOTHING".
	// See https://stackoverflow.com/questions/34708509/how-to-use-returning-with-on-conflict-in-postgresql
	sqlStatement := fmt.Sprintf(`INSERT INTO %s VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING RETURNING id`, tableAWSResources) // nolint

	tagsBytes, _ := json.Marshal(cloudAssetChanges.Tags) // an error here is not possible considering json.Marshal is taking a simple map or nil

	if _, err := tx.ExecContext(ctx, sqlStatement, cloudAssetChanges.ARN, cloudAssetChanges.AccountID, cloudAssetChanges.Region, cloudAssetChanges.ResourceType, tagsBytes); err != nil {
		return err
	}

	return nil
}

func (db *DB) recordNetworkChanges(ctx context.Context, resourceID string, timestamp time.Time, changes domain.NetworkChanges, tx *sql.Tx) error {

	for _, hostname := range changes.Hostnames {
		err := db.insertHostname(ctx, hostname, tx)
		if err != nil {
			return err
		}
	}

	for _, val := range changes.PrivateIPAddresses {
		isJoin := false
		if strings.EqualFold(added, changes.ChangeType) {
			isJoin = true
		}
		err := db.insertIPAddress(ctx, val, tx)
		if err != nil {
			return err
		}
		err = db.insertNetworkChangeEvent(ctx, timestamp, false, isJoin, resourceID, val, nil, tx)
		if err != nil {
			return err
		}
	}

	for _, val := range changes.PublicIPAddresses {
		isJoin := false
		if strings.EqualFold(added, changes.ChangeType) {
			isJoin = true
		}
		err := db.insertIPAddress(ctx, val, tx)
		if err != nil {
			return err
		}
		// yeah, a nested for loop to establish the relationship of every public IP to every hostname
		for _, hostname := range changes.Hostnames {
			err := db.insertNetworkChangeEvent(ctx, timestamp, true, isJoin, resourceID, val, &hostname, tx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (db *DB) insertIPAddress(ctx context.Context, ipAddress string, tx *sql.Tx) error {

	// this lib won't give back the last INSERTed row ID, so we don't bother with `RETURNING ...`
	// See https://stackoverflow.com/questions/34708509/how-to-use-returning-with-on-conflict-in-postgresql
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s VALUES ($1) ON CONFLICT DO NOTHING`, tableAWSIPS), ipAddress); err != nil { // nolint
		return err
	}

	return nil
}

func (db *DB) insertHostname(ctx context.Context, hostname string, tx *sql.Tx) error {

	// this lib won't give back the last INSERTed row ID, so we don't bother with `RETURNING ...`
	// See https://stackoverflow.com/questions/34708509/how-to-use-returning-with-on-conflict-in-postgresql
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s VALUES ($1) ON CONFLICT DO NOTHING`, tableAWSHostnames), hostname); err != nil { // nolint
		return err
	}

	return nil
}

func (db *DB) insertNetworkChangeEvent(ctx context.Context, timestamp time.Time, isPublic bool, isJoin bool, resourceID string, ipAddress string, hostname *string, tx *sql.Tx) error {
	// Postgres 11 automatically propagates the parent index to child tables, so no need to
	// explicitly create an index on the possibly created new table.

	// this lib won't give back the last INSERTed row ID, so we don't bother with `RETURNING ...`
	// See https://stackoverflow.com/questions/34708509/how-to-use-returning-with-on-conflict-in-postgresql
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s VALUES ($1, $2, $3, $4, $5, $6)`, tableAWSEventsIPSHostnames), timestamp, isPublic, isJoin, resourceID, ipAddress, hostname) // nolint
	return err
}

// FetchAll gets all the assets present at the specified time
func (db *DB) FetchAll(ctx context.Context, when time.Time, count uint, offset uint, typeFilter string) ([]domain.CloudAssetDetails, error) {
	return db.runQuery(ctx, 1, bulkResourcesQuery, when, typeFilter, count, offset)
}

// FetchByHostname gets the assets who have hostname at the specified time
func (db *DB) FetchByHostname(ctx context.Context, when time.Time, hostname string) ([]domain.CloudAssetDetails, error) {
	sqlstmt := fmt.Sprintf(latestStatusQuery, `aws_hostnames_hostname`)
	return db.runQuery(ctx, 1, sqlstmt, hostname, when)
}

// FetchByIP gets the assets who have IP address at the specified time
func (db *DB) FetchByIP(ctx context.Context, when time.Time, ipAddress string) ([]domain.CloudAssetDetails, error) {
	ver, err := db.GetSchemaVersion(ctx)
	if err != nil {
		return nil, err
	}
	var asset []domain.CloudAssetDetails
	if ver < DualWriteSchemaVersion {
		sqlstmt := fmt.Sprintf(latestStatusQuery, `aws_ips_ip`)
		asset, err = db.runQuery(ctx, ver, sqlstmt, ipAddress, when)
	} else {
		sqlstmt2 := fmt.Sprintf(resourceByPublicIPQuery + " ")
		asset, err = db.runQuery(ctx, ver, sqlstmt2, ipAddress, when)
	}
	return asset, err
}

// runQuery helps to get cloud asset details by running database query with argument(s)
func (db *DB) runQuery(ctx context.Context, version uint, query string, args ...interface{}) ([]domain.CloudAssetDetails, error) {

	rows, err := db.sqldb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	cloudAssetDetails := make([]domain.CloudAssetDetails, 0)

	tempMap := make(map[string]*domain.CloudAssetDetails)

	for rows.Next() {
		var row domain.CloudAssetDetails
		var metaBytes []byte
		var hostname sql.NullString
		var ipAddress string // no need for sql.NullBool as the DB column is guaranteed a value
		var isPublic bool    // no need for sql.NullBool as the DB column is guaranteed a value
		var isJoin bool      // no need for sql.NullBool as the DB column is guaranteed a value
		var timestamp time.Time

		if version < DualWriteSchemaVersion {
			err = rows.Scan(&row.ARN, &ipAddress, &hostname, &isPublic, &isJoin, &timestamp, &row.AccountID, &row.Region, &row.ResourceType, &metaBytes)
		} else {
			err = rows.Scan(&ipAddress, &hostname, &row.ARN, &metaBytes, &row.Region, &row.ResourceType, &row.AccountID)
			isPublic = true
		}

		if err != nil {
			return nil, err
		}

		if metaBytes != nil {
			var i map[string]string
			_ = json.Unmarshal(metaBytes, &i) // we already checked for nil, and the DB column is JSONB; no need for err check here
			row.Tags = i
		}
		if tempMap[row.ARN] == nil {
			tempMap[row.ARN] = &row
		}
		found := false
		if hostname.Valid {
			for _, val := range tempMap[row.ARN].Hostnames {
				if strings.EqualFold(val, hostname.String) {
					found = true
					break
				}
			}
			if !found {
				tempMap[row.ARN].Hostnames = append(tempMap[row.ARN].Hostnames, hostname.String)
			}
		}
		found = false
		var ipAddresses *[]string
		if isPublic {
			ipAddresses = &tempMap[row.ARN].PublicIPAddresses
		} else {
			ipAddresses = &tempMap[row.ARN].PrivateIPAddresses
		}
		for _, val := range *ipAddresses {
			if strings.EqualFold(val, ipAddress) {
				found = true
				break
			}
		}
		if !found {
			newArray := append(*ipAddresses, ipAddress)
			if isPublic {
				tempMap[row.ARN].PublicIPAddresses = newArray
			} else {
				tempMap[row.ARN].PrivateIPAddresses = newArray
			}
		}
	}

	rows.Close() // no need to capture the returned error since we check rows.Err() immediately:

	if err = rows.Err(); err != nil {
		return nil, err
	}

	for _, val := range tempMap {
		cloudAssetDetails = append(cloudAssetDetails, *val)
	}

	return cloudAssetDetails, nil
}

func (db *DB) assignPrivateIP(ctx context.Context, tx *sql.Tx, arnID string, ip string, when time.Time) error {
	const assignPrivateIPQuery = `
do
$$
    begin
        update aws_private_ip_assignment
        set not_before = $1
        where private_ip = $2
          and not_before = to_timestamp(0)
          and not_after > $1
          and aws_resource_id = (select id from aws_resource where arn_id = $3);
        if not found then
            insert into aws_private_ip_assignment
                (not_before, private_ip, aws_resource_id)
            values ($1, $2, (select id from aws_resource where arn_id = $3));
        end if;
    end
$$;
`
	_, err := tx.ExecContext(ctx, assignPrivateIPQuery, when, ip, arnID)
	return err
}

func (db *DB) releasePrivateIP(ctx context.Context, tx *sql.Tx, arnID string, ip string, when time.Time) error {
	//we use to_timestamp(0) for release events w/o known assignment as the start of epoch - 1970-01-01 is known
	//to not have AWS resources by definition
	//this way we can find unbalanced events while avoiding nullable not_before
	//which provides minimal support for out-of-order events
	const releasePrivateIPQuery = `
do
$$
    begin
        update aws_private_ip_assignment
        set not_after=$1
        where private_ip = $2
          and aws_resource_id = (select id from aws_resource where arn_id = $3);
        if not found then
            insert into aws_private_ip_assignment
                (not_before, not_after, private_ip, aws_resource_id)
            values (to_timestamp(0), $1, $2, (select id from aws_resource where arn_id = $3));
        end if;
    end
$$;
`
	_, err := tx.ExecContext(ctx, releasePrivateIPQuery, when, ip, arnID)
	return err
}

func (db *DB) assignPublicIP(ctx context.Context, tx *sql.Tx, arnID string, ip string, hostname string, when time.Time) error {
	const assignPublicIPQuery = `
do
$$
    begin
        update aws_public_ip_assignment
        set not_before = $1
        where public_ip = $2
          and not_before = to_timestamp(0)
          and not_after > $1
          and aws_resource_id = (select id from aws_resource where arn_id = $3)
          and aws_hostname = $4;
        if not found then
            insert into aws_public_ip_assignment
                (not_before, public_ip, aws_resource_id, aws_hostname)
            values ($1, $2, (select id from aws_resource where arn_id = $3), $4);
        end if;
    end
$$;
`
	_, err := tx.ExecContext(ctx, assignPublicIPQuery, when, ip, arnID, hostname)
	return err
}

func (db *DB) releasePublicIP(ctx context.Context, tx *sql.Tx, arnID string, ip string, hostname string, when time.Time) error {
	//we use to_timestamp(0) for release events w/o known assignment as the start of epoch - 1970-01-01 is known
	//to NOT have AWS resources by definition
	//this way we can find unbalanced events while avoiding nullable not_before
	//which provides minimal support for out-of-order events
	const releasePublicIPQuery = `
do
$$
    begin
        update aws_public_ip_assignment
        set not_after=$1
        where public_ip = $2
          and aws_resource_id = (select id from aws_resource where arn_id = $3)
          and aws_hostname = $4;
        if not found then
            insert into aws_public_ip_assignment
                (not_before, not_after, public_ip, aws_resource_id, aws_hostname)
            values (to_timestamp(0), $1, $2, (select id from aws_resource where arn_id = $3), $4);
        end if;
    end
$$;
`
	_, err := tx.ExecContext(ctx, releasePublicIPQuery, when, ip, arnID, hostname)
	return err
}

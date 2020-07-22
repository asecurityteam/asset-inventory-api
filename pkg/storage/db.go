package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

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

	added   = "ADDED"   // one of the network event types we track
	deleted = "DELETED" // deleted network event

)

// can't use Sprintf in a const, so...
// %s should be `aws_hostnames_hostname` or `aws_ips_ip`
// nolint
const latestStatusQuery = `WITH latest_candidates AS (
	    SELECT
	        *,
	        MAX(ts) OVER (PARTITION BY aws_events_ips_hostnames.aws_resources_id) as max_ts
	    FROM aws_events_ips_hostnames
	    WHERE
	        aws_events_ips_hostnames.%s = $1 AND
	        aws_events_ips_hostnames.ts <= $2
	),
	latest AS (
	    SELECT *
	    FROM latest_candidates
	    WHERE
	        latest_candidates.ts = latest_candidates.max_ts AND
	        latest_candidates.is_join = 'true'
	)
	SELECT
	    latest.aws_resources_id,
	    latest.aws_ips_ip,
	    latest.aws_hostnames_hostname,
	    latest.is_public,
	    latest.is_join,
	    latest.ts,
	    aws_resources.account_id,
	    aws_resources.region,
	    aws_resources.type,
	    aws_resources.meta
	FROM latest
	    LEFT OUTER JOIN
	    aws_resources ON
	        latest.aws_resources_id = aws_resources.id;
`

// Query to find resource by private IP using v2 schema
const resourceByPrivateIPQuery = `select * from get_resource_by_private_ip($1, $2)`

// Query to find resource by public IP using v2 schema
const resourceByPublicIPQuery = `select * from get_resource_by_public_ip($1, $2)`

// Query to find resource by hostname using v2 schema
const resourceByHostnameQuery = `select * from get_resource_by_hostname($1, $2)`

// Query to find resource by ARN ID
const resourceByARNIDQuery = `select * from get_resource_by_arn_id($1, $2)`

// Query to find owner and champions by account ID, which is auto-increment primary key
const ownerByAccountIDQuery = `select * from get_owner_and_champions_by_account_id($1)`

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

const insertPersonQuery = `
INSERT INTO person(login, email, name, valid)
VALUES ($1, $2, $3, $4)
ON CONFLICT(login) DO UPDATE
SET email=$2, name=$3, valid=$4;
`

//TODO Optimized query to retrieve all the 'active' resources utilizing v2 schema. Out of scope currently.

// DB represents a convenient database abstraction layer
type DB struct {
	sqldb               *sql.DB // this is a unit test seam
	once                sync.Once
	now                 func() time.Time // unit test seam
	defaultPartitionTTL int
}

var privateIPNetworks = []net.IPNet{
	{
		IP:   net.IPv4(192, 168, 1, 0),
		Mask: net.IPv4Mask(255, 255, 0, 0),
	},
	{
		IP:   net.IPv4(172, 16, 0, 0),
		Mask: net.IPv4Mask(255, 240, 0, 0),
	},
	{
		IP:   net.IPv4(10, 0, 0, 0),
		Mask: net.IPv4Mask(255, 0, 0, 0),
	},
}

// Init initializes a connection to a Postgres database according to the environment variables POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DATABASE
func (db *DB) Init(ctx context.Context, url string, partitionTTL int) error {
	var initErr error
	db.once.Do(func() {

		db.defaultPartitionTTL = partitionTTL

		if db.now == nil {
			db.now = time.Now
		}

		if db.sqldb == nil {
			pgdb, err := sql.Open("postgres", url)
			if err != nil {
				initErr = err
				return // from the unnamed once.Do function
			}

			db.sqldb = pgdb
			err = db.ping()
			if err != nil {
				initErr = err
				return // from the unnamed once.Do function
			}

		}
	})
	return initErr
}

// nb this is intentionally private for use by DB code to avoid confusion with fully-functioning SchemaVersionManager
func (db *DB) getSchemaVersion(ctx context.Context) (uint, error) {
	const versionQuery = `select version from schema_migrations`
	ver := uint(0)
	err := db.sqldb.QueryRowContext(ctx, versionQuery).Scan(&ver)
	if err != nil && err != sql.ErrNoRows { //no rows as version 0
		return 0, err
	}
	return ver, nil
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
	ver, err := db.getSchemaVersion(ctx)
	if err != nil || ver < DualWritesSchemaVersion { // the deployment pre-dates schema management or has old schema
		return db.StoreV1(ctx, cloudAssetChanges)
	}
	v2err := db.StoreV2(ctx, cloudAssetChanges)
	if ver < NewSchemaOnlyVersion { // need dual-write
		err = db.StoreV1(ctx, cloudAssetChanges)
		if err != nil {
			if v2err != nil { //both v1 and v2 failed, need to combine
				return fmt.Errorf("error storing in legacy schema (%v); error storing in new schema (%v)", err, v2err)
			}
			return err
		}
	}
	return v2err
}

// StoreV1 Storage interface implementation that records to database using legacy schema
func (db *DB) StoreV1(ctx context.Context, cloudAssetChanges domain.CloudAssetChanges) error {
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
	return nil
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
			for _, ip := range val.PublicIPAddresses {
				for _, hostname := range val.Hostnames { //TODO look very closely into awsconfig-tranformerd logic for this
					if strings.EqualFold(added, val.ChangeType) {
						err = db.assignPublicIP(ctx, tx, arnID, ip, hostname, cloudAssetChanges.ChangeTime)
					} else {
						err = db.releasePublicIP(ctx, tx, arnID, ip, hostname, cloudAssetChanges.ChangeTime)
					}
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
             values ($1::text, $2::text, $3::text, $4::text, $5::jsonb)
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

// extract unique resource-type/resource-id from full ARN format
// it is always the part after account-id:
// https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
func resIDFromARN(ARN string) string {
	parts := strings.SplitN(ARN, ":", 6)
	resourceID := parts[len(parts)-1]
	if strings.HasPrefix(resourceID, "loadbalancer/app") {
		return resourceID[13:]
	}
	parts = strings.SplitAfterN(resourceID, "/", -1)
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
	return db.runQuery(ctx, bulkResourcesQuery, when, typeFilter, count, offset)
}

// FetchByHostname gets the assets who have hostname at the specified time
func (db *DB) FetchByHostname(ctx context.Context, when time.Time, hostname string) ([]domain.CloudAssetDetails, error) {
	ver, err := db.getSchemaVersion(ctx)
	if err != nil {
		return nil, err
	}
	var assets []domain.CloudAssetDetails
	if ver < ReadsFromNewSchemaVersion {
		sqlstmt := fmt.Sprintf(latestStatusQuery, `aws_hostnames_hostname`)
		assets, err = db.runQuery(ctx, sqlstmt, hostname, when)
	} else {
		assets, err = db.runFetchByIPQuery(ctx, false, resourceByHostnameQuery, hostname, when)
	}
	return assets, err
}

// FetchByIP gets the assets who have IP address at the specified time
func (db *DB) FetchByIP(ctx context.Context, when time.Time, ipAddress string) ([]domain.CloudAssetDetails, error) {
	ver, err := db.getSchemaVersion(ctx)
	if err != nil {
		return nil, err
	}
	var asset []domain.CloudAssetDetails
	if ver < ReadsFromNewSchemaVersion {
		sqlstmt := fmt.Sprintf(latestStatusQuery, `aws_ips_ip`)
		asset, err = db.runQuery(ctx, sqlstmt, ipAddress, when)
	} else {
		ipaddr := net.ParseIP(ipAddress)
		if ipaddr == nil {
			return nil, errors.New("invalid IP address")
		}
		if isPrivateIP(ipaddr) {
			asset, err = db.runFetchByIPQuery(ctx, true, resourceByPrivateIPQuery, ipAddress, when)
		} else {
			asset, err = db.runFetchByIPQuery(ctx, false, resourceByPublicIPQuery, ipAddress, when)
		}
	}
	return asset, err
}

func isPrivateIP(ip net.IP) bool {
	for _, net := range privateIPNetworks {
		if net.Contains(ip) {
			return true
		}
	}
	return false
}

func (db *DB) runFetchByIPQuery(ctx context.Context, isPrivateIP bool, query string, args ...interface{}) ([]domain.CloudAssetDetails, error) {
	rows, err := db.sqldb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cloudAssetDetails := make([]domain.CloudAssetDetails, 0)
	tempMap := make(map[string]*domain.CloudAssetDetails)
	for rows.Next() {
		var row domain.CloudAssetDetails
		var account domain.AccountOwner

		var metaBytes []byte
		var hostname sql.NullString
		var ipAddress string // no need for sql.NullBool as the DB column is guaranteed a value
		var accountID int
		var chLogin string
		var chEmail string
		var chName string
		var chValid bool

		if isPrivateIP {
			err = rows.Scan(&ipAddress, &row.ARN, &metaBytes, &row.Region, &row.ResourceType, &row.AccountID,
				&accountID, &account.AccountID, &account.Owner.Login, &account.Owner.Email,
				&account.Owner.Name, &account.Owner.Valid, &chLogin, &chEmail, &chName, &chValid)
		} else {
			err = rows.Scan(&ipAddress, &hostname, &row.ARN, &metaBytes, &row.Region, &row.ResourceType, &row.AccountID,
				&accountID, &account.AccountID, &account.Owner.Login, &account.Owner.Email,
				&account.Owner.Name, &account.Owner.Valid, &chLogin, &chEmail, &chName, &chValid)
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

		if tempMap[row.ARN].AccountOwner.Champions == nil {
			tempMap[row.ARN].AccountOwner = domain.AccountOwner{
				AccountID: account.AccountID,
				Owner: domain.Person{
					Name:  account.Owner.Name,
					Login: account.Owner.Login,
					Email: account.Owner.Email,
					Valid: account.Owner.Valid,
				},
				Champions: make([]domain.Person, 0),
			}
		}

		champion := domain.Person{
			Login: chLogin,
			Email: chEmail,
			Name:  chName,
			Valid: chValid,
		}

		found := false
		for _, val := range tempMap[row.ARN].AccountOwner.Champions {
			if strings.EqualFold(val.Email, chEmail) {
				found = true
				break
			}
		}
		if !found {
			tempMap[row.ARN].AccountOwner.Champions = append(tempMap[row.ARN].AccountOwner.Champions, champion)
		}

		found = false
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
		if !isPrivateIP {
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
			if !isPrivateIP {
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

// FetchByARNID gets the assets who have ARN ID at the specified time
func (db *DB) FetchByARNID(ctx context.Context, when time.Time, arnID string) ([]domain.CloudAssetDetails, error) {
	ver, err := db.getSchemaVersion(ctx)
	if err != nil {
		return nil, err
	}
	if ver < ReadsFromNewSchemaVersion {
		return nil, nil
	}

	rows, err := db.sqldb.QueryContext(ctx, resourceByARNIDQuery, arnID, when)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cloudAssetDetails := make([]domain.CloudAssetDetails, 0)

	tempPublicIPMap := make(map[string]struct{})
	tempPrivateIPMap := make(map[string]struct{})
	tempHostnameMap := make(map[string]struct{})
	tempChampionsMap := make(map[string]domain.Person)

	var account domain.AccountOwner
	var asset domain.CloudAssetDetails
	var accountID int
	hasTag := false
	empty := true
	for rows.Next() {
		empty = false // there is no way to check for size of SQL result set in golang
		var privateIPAddress sql.NullString
		var publicIPAddress sql.NullString
		var hostname sql.NullString
		var metaBytes []byte
		var chLogin string
		var chEmail string
		var chName string
		var chValid bool
		if err = rows.Scan(&privateIPAddress, &publicIPAddress, &hostname, &asset.ResourceType, &asset.AccountID,
			&asset.Region, &metaBytes, &accountID, &account.AccountID, &account.Owner.Login, &account.Owner.Email,
			&account.Owner.Name, &account.Owner.Valid, &chLogin, &chEmail, &chName, &chValid); err != nil {
			return nil, err
		}

		if privateIPAddress.Valid {
			tempPrivateIPMap[privateIPAddress.String] = struct{}{}
		}
		if publicIPAddress.Valid {
			tempPublicIPMap[publicIPAddress.String] = struct{}{}
			if hostname.Valid {
				tempHostnameMap[hostname.String] = struct{}{}
			}
		}

		if metaBytes != nil && !hasTag {
			var i map[string]string
			_ = json.Unmarshal(metaBytes, &i) // we already checked for nil, and the DB column is JSONB; no need for err check here
			asset.Tags = i
			hasTag = true
		}

		tempChampionsMap[chLogin+chEmail] = domain.Person{
			Login: chLogin,
			Email: chEmail,
			Name:  chName,
			Valid: chValid,
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if empty { // we got 0 rows, nothing to process
		return cloudAssetDetails, nil
	}

	for ip := range tempPrivateIPMap {
		asset.PrivateIPAddresses = append(asset.PrivateIPAddresses, ip)
	}
	for ip := range tempPublicIPMap {
		asset.PublicIPAddresses = append(asset.PublicIPAddresses, ip)
	}
	for hostname := range tempHostnameMap {
		asset.Hostnames = append(asset.Hostnames, hostname)
	}
	asset.ARN = arnID

	asset.AccountOwner = domain.AccountOwner{
		AccountID: account.AccountID,
		Owner: domain.Person{
			Name:  account.Owner.Name,
			Login: account.Owner.Login,
			Email: account.Owner.Email,
			Valid: account.Owner.Valid,
		},
		Champions: make([]domain.Person, 0),
	}
	for champion := range tempChampionsMap {
		asset.AccountOwner.Champions = append(asset.AccountOwner.Champions, tempChampionsMap[champion])
	}
	cloudAssetDetails = append(cloudAssetDetails, asset)
	return cloudAssetDetails, err
}

// FetchAccountOwnerByID fetches account owner and champions with account ID
func (db *DB) FetchAccountOwnerByID(ctx context.Context, query string, accountID int) (domain.AccountOwner, error) {

	rows, err := db.sqldb.QueryContext(ctx, query, accountID)

	if err != nil {
		return domain.AccountOwner{}, err
	}

	defer rows.Close()

	champions := make([]domain.Person, 0)
	var account domain.AccountOwner
	for rows.Next() {
		var chLogin string
		var chEmail string
		var chName string
		var chValid bool
		err = rows.Scan(&account.AccountID, &account.Owner.Login, &account.Owner.Email, &account.Owner.Name, &account.Owner.Valid, &chLogin, &chEmail, &chName, &chValid)
		if err != nil {
			return domain.AccountOwner{}, err
		}

		champions = append(champions, domain.Person{
			Login: chLogin,
			Email: chEmail,
			Name:  chName,
			Valid: chValid,
		})
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return domain.AccountOwner{}, err
	}
	account.Champions = champions
	return account, nil
}

func (db *DB) runQuery(ctx context.Context, query string, args ...interface{}) ([]domain.CloudAssetDetails, error) {

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

		err = rows.Scan(&row.ARN, &ipAddress, &hostname, &isPublic, &isJoin, &timestamp, &row.AccountID, &row.Region, &row.ResourceType, &metaBytes)

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
	const assignPrivateIPQueryUpdate = `
update aws_private_ip_assignment
set not_before = $1
where private_ip = $2
  and not_before = to_timestamp(0)
  and not_after > $1
  and aws_resource_id = (select id from aws_resource where arn_id = $3);`

	const assignPrivateIPQueryInsert = `
insert into aws_private_ip_assignment
    (not_before, private_ip, aws_resource_id)
values ($1, $2, (select id from aws_resource where arn_id = $3)) on conflict do nothing ;`

	res, err := tx.ExecContext(ctx, assignPrivateIPQueryUpdate, when, ip, arnID)
	if err != nil {
		return err
	}
	changedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changedRows != 0 {
		return nil
	}
	_, err = tx.ExecContext(ctx, assignPrivateIPQueryInsert, when, ip, arnID)
	return err
}

func (db *DB) releasePrivateIP(ctx context.Context, tx *sql.Tx, arnID string, ip string, when time.Time) error {
	//we use to_timestamp(0) for release events w/o known assignment as the start of epoch - 1970-01-01 is known
	//to not have AWS resources by definition
	//this way we can find unbalanced events while avoiding nullable not_before
	//which provides minimal support for out-of-order events
	const releasePrivateIPQueryUpdate = `
update aws_private_ip_assignment
set not_after=$1
where private_ip = $2
  and aws_resource_id = (select id from aws_resource where arn_id = $3);`

	const releasePrivateIPQueryInsert = `
insert into aws_private_ip_assignment
    (not_before, not_after, private_ip, aws_resource_id)
values (to_timestamp(0), $1, $2, (select id from aws_resource where arn_id = $3)) on conflict do nothing ;`

	res, err := tx.ExecContext(ctx, releasePrivateIPQueryUpdate, when, ip, arnID)
	if err != nil {
		return err
	}
	changedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changedRows != 0 {
		return nil
	}
	_, err = tx.ExecContext(ctx, releasePrivateIPQueryInsert, when, ip, arnID)
	return err
}

func (db *DB) assignPublicIP(ctx context.Context, tx *sql.Tx, arnID string, ip string, hostname string, when time.Time) error {
	const assignPublicIPQueryUpdate = `
update aws_public_ip_assignment
set not_before = $1
where public_ip = $2
  and not_before = to_timestamp(0)
  and not_after > $1
  and aws_resource_id = (select id from aws_resource where arn_id = $3)
  and aws_hostname = $4`

	const assignPublicIPQueryInsert = `
insert into aws_public_ip_assignment
    (not_before, public_ip, aws_resource_id, aws_hostname)
values ($1, $2, (select id from aws_resource where arn_id = $3), $4) on conflict do nothing`

	res, err := tx.ExecContext(ctx, assignPublicIPQueryUpdate, when, ip, arnID, hostname)
	if err != nil {
		return err
	}
	changedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changedRows != 0 {
		return nil
	}
	_, err = tx.ExecContext(ctx, assignPublicIPQueryInsert, when, ip, arnID, hostname)
	return err
}

func (db *DB) releasePublicIP(ctx context.Context, tx *sql.Tx, arnID string, ip string, hostname string, when time.Time) error {
	//we use to_timestamp(0) for release events w/o known assignment as the start of epoch - 1970-01-01 is known
	//to NOT have AWS resources by definition
	//this way we can find unbalanced events while avoiding nullable not_before
	//which provides minimal support for out-of-order events
	const releasePublicIPQueryUpdate = `
        update aws_public_ip_assignment
        set not_after=$1
        where public_ip = $2
          and aws_resource_id = (select id from aws_resource where arn_id = $3)
          and aws_hostname = $4
          and not_after is null`

	const releasePublicIPQueryInsert = `
            insert into aws_public_ip_assignment
                (not_before, not_after, public_ip, aws_resource_id, aws_hostname)
            values (to_timestamp(0), $1, $2, (select id from aws_resource where arn_id = $3), $4) on conflict do nothing `

	res, err := tx.ExecContext(ctx, releasePublicIPQueryUpdate, when, ip, arnID, hostname)
	if err != nil {
		return err
	}
	changedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changedRows != 0 {
		return nil
	}
	_, err = tx.ExecContext(ctx, releasePublicIPQueryInsert, when, ip, arnID, hostname)
	return err
}

// BackFillEventsLocally launches the bulk back-fill process using local write handler
func (db *DB) BackFillEventsLocally(ctx context.Context, from time.Time, to time.Time) error {
	return db.exportEvents(ctx, from, to, NewLocalExportHandler(db))
}

func (db *DB) exportEvents(ctx context.Context, notBefore time.Time, notAfter time.Time, handler domain.EventExportHandler) error {
	const exportEventsQuery = `
select ae.ts,
       ae.aws_resources_id,
       ar.type,
       ar.region,
       ar.account_id,
       ar.meta,
       ae.aws_ips_ip,
       ae.aws_hostnames_hostname,
       ae.is_join,
       ae.is_public
from aws_events_ips_hostnames as ae
         left join aws_resources ar on ae.aws_resources_id = ar.id
where ae.ts >= $1
  and ae.ts <= $2
order by ae.ts asc
`
	rows, err := db.sqldb.QueryContext(ctx, exportEventsQuery, notBefore, notAfter)
	if err != nil {
		return err
	}

	defer rows.Close()
	for rows.Next() {
		var ts time.Time
		var arn string
		var resourceType string
		var region string
		var accountID string
		var meta []byte
		var ip string
		var hostname sql.NullString
		var isJoin bool   // no need for sql.NullBool as the DB column is guaranteed a value
		var isPublic bool // no need for sql.NullBool as the DB column is guaranteed a value
		err = rows.Scan(&ts, &arn, &resourceType, &region, &accountID, &meta, &ip, &hostname, &isJoin, &isPublic)
		if err != nil {
			return err
		}
		chg := domain.NetworkChanges{
			ChangeType: deleted,
		}
		if isJoin {
			chg.ChangeType = added
		}

		if isPublic {
			chg.Hostnames = []string{hostname.String}
			chg.PublicIPAddresses = []string{ip}
		} else {
			chg.PrivateIPAddresses = []string{ip}
		}

		changes := domain.CloudAssetChanges{
			Changes:      []domain.NetworkChanges{chg},
			ChangeTime:   ts,
			ResourceType: resourceType,
			AccountID:    accountID,
			Region:       region,
			ARN:          arn,
		}
		// copy/paste from existing runQuery, might need to move this out to a helper
		if meta != nil {
			var i map[string]string
			_ = json.Unmarshal(meta, &i) // we already checked for nil, and the DB column is JSONB; no need for err check here
			changes.Tags = i
		}
		err = handler.Handle(changes)
		if err != nil {
			return err
		}
	}
	return nil
}

// StoreAccountOwner is an implementation of AccountOwnerStorer interface that saves account ID, its owner and champions of the account to a database
func (db *DB) StoreAccountOwner(ctx context.Context, accountOwner domain.AccountOwner) error {

	tx, err := db.sqldb.Begin()
	if err != nil {
		return err
	}

	err = db.storeAccountOwner(ctx, accountOwner, tx)

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

func (db *DB) storeAccountOwner(ctx context.Context, accountOwner domain.AccountOwner, tx *sql.Tx) error {

	sqlStatement := `
			INSERT INTO aws_account (account)
			VALUES ($1)
			ON CONFLICT DO NOTHING
			`
	var err error
	if _, err = tx.ExecContext(ctx, sqlStatement, accountOwner.AccountID); err != nil {
		return err
	}

	// Insert or update details of account owner
	if _, err = tx.ExecContext(ctx, insertPersonQuery, accountOwner.Owner.Login, accountOwner.Owner.Email, accountOwner.Owner.Name, accountOwner.Owner.Valid); err != nil {
		return err
	}

	sqlStatement = `SELECT id FROM person WHERE login=$1`
	row := tx.QueryRowContext(ctx, sqlStatement, accountOwner.Owner.Login)
	var personID int
	if err := row.Scan(&personID); err != nil {
		return err
	}

	sqlStatement = `SELECT id FROM aws_account WHERE account=$1`
	row = tx.QueryRowContext(ctx, sqlStatement, accountOwner.AccountID)
	var accountID int
	if err := row.Scan(&accountID); err != nil {
		return err
	}

	sqlStatement = `
			INSERT INTO account_owner
			VALUES ($1, $2)
			ON CONFLICT (aws_account_id)
			DO UPDATE SET person_id = $1, aws_account_id = $2
			`
	if _, err := tx.ExecContext(ctx, sqlStatement, personID, accountID); err != nil {
		return err
	}

	sqlStatement = `DELETE FROM account_champion WHERE aws_account_id=$1`
	if _, err := tx.ExecContext(ctx, sqlStatement, accountID); err != nil {
		return err
	}
	if len(accountOwner.Champions) > 0 {
		for _, person := range accountOwner.Champions {
			// Add champion to "person" table if champion does not exists in "person" table
			if _, err := tx.ExecContext(ctx, insertPersonQuery, person.Login, person.Email, person.Name, person.Valid); err != nil {
				return err
			}

			row := tx.QueryRowContext(ctx, `SELECT id FROM person WHERE login=$1`, person.Login)
			var champID int
			if err := row.Scan(&champID); err != nil {
				return err
			}
			sqlStatement = `
					INSERT INTO account_champion(person_id, aws_account_id)
					VALUES ($1, $2)
					ON CONFLICT DO NOTHING
					`
			if _, err := tx.ExecContext(ctx, sqlStatement, champID, accountID); err != nil {
				return err
			}
		}
	}
	return nil
}

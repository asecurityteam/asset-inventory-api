package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	_ "github.com/lib/pq" // must remain here for sql lib to find the postgres driver
	"github.com/pkg/errors"
)

const (
	defaultPartitionInterval   = 3 // months
	tablePartitions            = "partitions"
	tableAWSResources          = "aws_resources"
	tableAWSIPS                = "aws_ips"
	tableAWSHostnames          = "aws_hostnames"
	tableAWSEventsIPSHostnames = "aws_events_ips_hostnames"
	createScript               = "2_create.sql"

	added = "ADDED" // one of the network event types we track
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

// DB represents a convenient database abstraction layer
type DB struct {
	sqldb   *sql.DB // this is a unit test seam
	scripts func(name string) (string, error)
	once    sync.Once
	now     func() time.Time // unit test seam
}

// RunScript executes a SQL script from disk against the database.
func (db *DB) RunScript(ctx context.Context, name string) error {
	scriptContent, err := db.scripts(name)
	if err != nil {
		return err
	}
	tx, err := db.sqldb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, scriptContent); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to rollback from %s because of %s", err.Error(), rbErr.Error())
		}
		return err
	}
	return tx.Commit()
}

// Init initializes a connection to a Postgres database according to the environment variables POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DATABASE
func (db *DB) Init(ctx context.Context, postgresConfig *PostgresConfig) error {
	var initerr error
	db.once.Do(func() {

		host := postgresConfig.Hostname
		port := postgresConfig.Port
		user := postgresConfig.Username
		password := postgresConfig.Password
		dbname := postgresConfig.DatabaseName

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
			psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
				"password=%s dbname=%s sslmode=%s",
				host, port, user, password, "postgres", sslmode)
			pgdb, err := sql.Open("postgres", psqlInfo)
			if err != nil {
				initerr = err
				return // from the unnamed once.Do function
			}

			db.sqldb = pgdb

			dbExists, err := db.doesDBExist(dbname)
			if err != nil {
				initerr = err
				return // from the unnamed once.Do function
			}

			if !dbExists {
				err = db.create(dbname)
				if err != nil {
					initerr = err
					return // from the unnamed once.Do function
				}
			}

			psqlInfo = fmt.Sprintf("host=%s port=%s user=%s "+
				"password=%s dbname=%s sslmode=%s",
				host, port, user, password, dbname, sslmode)
			err = db.use(psqlInfo)
			if err != nil {
				initerr = err
				return // from the unnamed once.Do function
			}

			err = db.ping()
			if err != nil {
				initerr = err
				return // from the unnamed once.Do function
			}

		}
		initerr = db.RunScript(ctx, createScript)
	})
	return initerr
}

func (db *DB) doesDBExist(dbName string) (bool, error) {
	row := db.sqldb.QueryRow("SELECT datname FROM pg_catalog.pg_database WHERE lower(datname) = lower($1);", dbName)
	var id string
	if err := row.Scan(&id); err != nil {
		switch err {
		case sql.ErrNoRows:
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

func (db *DB) create(name string) error {

	_, err := db.sqldb.Exec("CREATE DATABASE " + name + ";") // nolint
	if err != nil {
		return err
	}

	return nil
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

// GeneratePartition finds the latest partition, and generate the next partition based on the previous partions time range
func (db *DB) GeneratePartition(ctx context.Context) error {
	tx, err := db.sqldb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, fmt.Sprintf("LOCK TABLE %s", tablePartitions)); err != nil { // nolint
		return handleRollback(tx, err)
	}
	stmt := "SELECT partition_end FROM partitions ORDER BY partition_end DESC LIMIT 1"
	row := tx.QueryRowContext(ctx, stmt)
	var latest string
	err = row.Scan(&latest)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return generatePartitionForTime(ctx, tx, db.now(), db.now())
	default:
		return handleRollback(tx, err)
	}

	latestTime, err := time.Parse(time.RFC3339, latest)
	if err != nil {
		return handleRollback(tx, fmt.Errorf("Invalid partition range: %s, %v", latest, err))
	}

	return generatePartitionForTime(ctx, tx, db.now(), latestTime)
}

// GeneratePartitionWithTimestamp generates the partition based on the given time
func (db *DB) GeneratePartitionWithTimestamp(ctx context.Context, begin time.Time) error {
	tx, err := db.sqldb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, fmt.Sprintf("LOCK TABLE %s", tablePartitions)); err != nil { // nolint
		return handleRollback(tx, err)
	}

	return generatePartitionForTime(ctx, tx, db.now(), begin)
}

func generatePartitionForTime(ctx context.Context, tx *sql.Tx, createdAt, begin time.Time) error {
	monthInterval := defaultPartitionInterval
	begin = time.Date(begin.Year(), begin.Month(), 1, 0, 0, 0, 0, begin.Location()) // month granularity
	end := begin.AddDate(0, monthInterval, 0)
	partitionTableNameSuffix := fmt.Sprintf(`%d_%02dto%d_%02d`, begin.Year(), begin.Month(), end.Year(), end.Month())
	name := fmt.Sprintf(`%s_%s`, tableAWSEventsIPSHostnames, partitionTableNameSuffix)

	// nolint
	stmt := fmt.Sprintf(`INSERT INTO %s(name, created_at, partition_begin, partition_end)
    	SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
    		SELECT 1 FROM %s WHERE (
        		(partition_begin <= $3 AND partition_end > $3) OR
        		(partition_begin <= $4 AND partition_end > $4)
    		)
		)`, tablePartitions, tablePartitions)
	res, err := tx.ExecContext(ctx, stmt, name, createdAt, begin, end)
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
	return tx.Commit()

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

// FetchByHostname gets the assets who have hostname at the specified time
func (db *DB) FetchByHostname(ctx context.Context, when time.Time, hostname string) ([]domain.CloudAssetDetails, error) {
	sqlstmt := fmt.Sprintf(latestStatusQuery, `aws_hostnames_hostname`)
	return db.runQuery(ctx, sqlstmt, hostname, when)
}

// FetchByIP gets the assets who have IP address at the specified time
func (db *DB) FetchByIP(ctx context.Context, when time.Time, ipAddress string) ([]domain.CloudAssetDetails, error) {
	sqlstmt := fmt.Sprintf(latestStatusQuery, `aws_ips_ip`)
	return db.runQuery(ctx, sqlstmt, ipAddress, when)
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

		if err == nil {
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

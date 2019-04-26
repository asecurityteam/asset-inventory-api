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
	_ "github.com/lib/pq" // postgres driver for sql must be imported so sql finds and uses it
	"github.com/pkg/errors"
)

const tableAWSResources = "aws_resources"
const tableAWSIPS = "aws_ips"
const tableAWSHostnames = "aws_hostnames"
const tableAWSEventsIPSHostnames = "aws_events_ips_hostnames"

const added = "ADDED" // one of the network event types we track

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
	sqldb *sql.DB // this is a unit test seam
	once  sync.Once
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

		if db.sqldb == nil {
			sslmode := "disable"
			if host != "localhost" && host != "postgres" {
				sslmode = "require"
			}
			psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
				"password=%s dbname=%s sslmode=%s",
				host, port, user, password, dbname, sslmode)
			pgdb, err := sql.Open("postgres", psqlInfo)
			if err != nil {
				initerr = err
				return // from the unnamed once.Do function
			}

			err = pgdb.Ping()
			if err != nil {
				initerr = err
			}

			db.sqldb = pgdb
		}
	})
	return initerr
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

	tagsBytes, err := json.Marshal(cloudAssetChanges.Tags)
	if err != nil {
		tagsBytes = nil
	}

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

	// Postgres does not auto-create partition tables, so we do it.  We're using 3-month intervals (quarters)
	monthInterval := 3
	year := timestamp.Year()
	fromMonth := (int(timestamp.Month()-1)/monthInterval)*monthInterval + 1 // get the interval of the year we're in, then use the first month of that quarter
	fromDay := 1
	toMonth := (int(timestamp.Month()-1)/monthInterval)*monthInterval + monthInterval          // get the interval of the year we're in, then use the last month of that interval
	toDay := time.Date(year, time.Month(toMonth+1), 0, 0, 0, 0, 0, timestamp.Location()).Day() // time.Date will normalize that toMonth + 1 and 0 day shenanigans
	from := fmt.Sprintf(`%d-%02d-%02d`, year, fromMonth, fromDay)
	to := fmt.Sprintf(`%d-%02d-%02d`, year, toMonth, toDay)
	partitionTableNameSuffix := fmt.Sprintf(`%d_%02dto%02d`, year, fromMonth, toMonth)
	tableName := fmt.Sprintf(`%s_%s`, tableAWSEventsIPSHostnames, partitionTableNameSuffix)

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR `+ // nolint
		`VALUES `+
		`FROM (`+
		`'%s') `+
		`TO (`+
		`'%s'`+
		`);`, tableName, tableAWSEventsIPSHostnames, from, to)); err != nil {
		return err
	}

	// we might be using Postgres version 10, which does not automatically propagate indices, so we do it:
	indexName := fmt.Sprintf("%s_aws_ips_ip_ts_idx", tableName) // this is Postgres naming convention, which we don't really have to follow
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING BTREE (aws_ips_ip, ts);", indexName, tableName)); err != nil {
		return err
	}

	// this lib won't give back the last INSERTed row ID, so we don't bother with `RETURNING ...`
	// See https://stackoverflow.com/questions/34708509/how-to-use-returning-with-on-conflict-in-postgresql
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s VALUES ($1, $2, $3, $4, $5, $6)`, tableAWSEventsIPSHostnames), timestamp, isPublic, isJoin, resourceID, ipAddress, hostname); err != nil { // nolint
		return err
	}

	return nil
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

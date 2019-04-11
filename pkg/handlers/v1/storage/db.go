package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"

	_ "github.com/lib/pq" // postgres driver for sql must be imported so sql finds and uses it
)

/*
The table designs are as such:

CREATE TABLE aws_resources (
    id VARCHAR PRIMARY KEY,
    account_id VARCHAR NOT NULL,
    region VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    meta JSONB
);

-- We use these simple tables to preserve uniqueness and so we can add columns for additional
-- metadata when needed without polluting the aws_events_ips_hostnames star table:

CREATE TABLE aws_ips (
    ip INET PRIMARY KEY
);

CREATE TABLE aws_hostnames (
    hostname VARCHAR PRIMARY KEY
);

-- Notice "PARTITION BY" below.  We're using built-in partitioning.
-- See https://blog.timescale.com/scaling-partitioning-data-postgresql-10-explained-cd48a712a9a1/

CREATE TABLE aws_events_ips_hostnames (
    ts TIMESTAMP NOT NULL,
    is_public BOOLEAN NOT NULL,
    is_join BOOLEAN NOT NULL,
    aws_resources_id VARCHAR NOT NULL,
    FOREIGN KEY (aws_resources_id) REFERENCES aws_resources (id),
    aws_ips_ip INET NOT NULL,
    FOREIGN KEY (aws_ips_ip) REFERENCES aws_ips (ip),
    aws_hostnames_hostname VARCHAR,
    FOREIGN KEY (aws_hostnames_hostname) REFERENCES aws_hostnames (hostname))
PARTITION BY
    RANGE (
        ts
);

-- And we'll make sure there's an index right away:

CREATE INDEX IF NOT EXISTS aws_events_ips_hostnames_aws_ips_ip_ts_idx ON aws_events_ips_hostnames USING BTREE (aws_ips_ip, ts);

Also, some good advice to follow:  https://www.vividcortex.com/blog/2015/09/22/common-pitfalls-go/

*/

const tableAWSResources = "aws_resources"
const tableAWSIPS = "aws_ips"
const tableAWSHostnames = "aws_hostnames"
const tableAWSEventsIPSHostnames = "aws_events_ips_hostnames"

// can't use Sprintf in a const, so...
const commonQuery = "SELECT " +
	"	aws_resources_id, aws_ips_ip, aws_hostnames_hostname, is_public, is_join, ts, aws_resources.account_id, aws_resources.region, aws_resources.type, aws_resources.meta " +
	"FROM " +
	"	" + tableAWSEventsIPSHostnames + " " +
	"JOIN " + tableAWSResources + " " +
	"	ON aws_resources_id = aws_resources.id " +
	"WHERE "

// DB represents a convenient database abstraction layer
type DB struct {
	sqldb *sql.DB      // this is a unit test seam
	LogFn domain.LogFn // also a seam

	once sync.Once
}

// Init initializes a connection to a Postgres database according to the environment variables POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DATABASE
func (db *DB) Init(ctx context.Context, postgresConfig *domain.PostgresConfig) error {
	var initerr error
	db.once.Do(func() {

		host := postgresConfig.Hostname
		port := postgresConfig.Port
		user := postgresConfig.Username
		password := postgresConfig.Password
		dbname := postgresConfig.DatabaseName

		logger := db.LogFn(ctx)

		if db.sqldb == nil {
			sslmode := "disable"
			if host != "localhost" {
				sslmode = "enable"
			}
			psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
				"password=%s dbname=%s sslmode=%s",
				host, port, user, password, dbname, sslmode)
			pgdb, err := sql.Open("postgres", psqlInfo)
			if err != nil {
				logger.Error(logs.DBError{Reason: err.Error()})
				initerr = err
			}

			err = pgdb.Ping()
			if err != nil {
				logger.Error(logs.DBError{Reason: err.Error()})
				initerr = err
			}

			logger.Info(logs.DBInfo{Reason: "Successfully connected to database"})

			db.sqldb = pgdb
		}
	})
	return initerr
}

// StoreCloudAsset an implementation of the Storage interface that records to a database
func (db *DB) StoreCloudAsset(ctx context.Context, cloudAssetChanges domain.CloudAssetChanges) error {
	tx, err := db.sqldb.Begin()
	if err != nil {
		return err
	}

	defer func() {
		switch err {
		case nil:
			err = tx.Commit()
		default:
			err = tx.Rollback()
		}
	}()

	if err = db.saveResource(ctx, cloudAssetChanges); err != nil {
		return err
	}

	for _, val := range cloudAssetChanges.Changes {
		err = db.recordNetworkChanges(ctx, cloudAssetChanges.ARN, cloudAssetChanges.ChangeTime, val, tx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) saveResource(ctx context.Context, cloudAssetChanges domain.CloudAssetChanges) error {
	// You won't get an ID back if nothing was done.  Also, this lib won't return the ID anyway even without the "ON CONFLICT DO NOTHING".
	// See https://stackoverflow.com/questions/34708509/how-to-use-returning-with-on-conflict-in-postgresql
	sqlStatement := fmt.Sprintf(`INSERT INTO %s VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING RETURNING id`, tableAWSResources)  // nolint

	if _, err := db.sqldb.ExecContext(ctx, sqlStatement, cloudAssetChanges.ARN, cloudAssetChanges.AccountID, cloudAssetChanges.Region, cloudAssetChanges.ResourceType, cloudAssetChanges.Tags); err != nil {
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
		if changes.ChangeType == "ADDED" {
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
		if changes.ChangeType == "ADDED" {
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
	fromMonth := (int((timestamp.Month())-1)/monthInterval)*monthInterval + 1 // get the interval of the year we're in, then use the first month of that quarter
	fromDay := 1
	toMonth := (int((timestamp.Month())-1)/monthInterval)*monthInterval + monthInterval        // get the interval of the year we're in, then use the last month of that interval
	toDay := time.Date(year, time.Month(toMonth+1), 0, 0, 0, 0, 0, timestamp.Location()).Day() // time.Date will normalize that toMonth + 1 and 0 day shenanigans
	from := fmt.Sprintf(`%d-%02d-%02d`, year, fromMonth, fromDay)
	to := fmt.Sprintf(`%d-%02d-%02d`, year, toMonth, toDay)
	partitionTableNameSuffix := fmt.Sprintf(`%d_%02dto%02d`, year, fromMonth, toMonth)
	tableName := fmt.Sprintf(`%s_%s`, tableAWSEventsIPSHostnames, partitionTableNameSuffix)

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR `+  // nolint
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

	// // this lib won't give back the last INSERTed row ID, so we don't bother with `RETURNING ...`
	// // See https://stackoverflow.com/questions/34708509/how-to-use-returning-with-on-conflict-in-postgresql
	// tsFormat := timestamp.Format("2006-01-02 15:04:05.123")
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s VALUES ($1, $2, $3, $4, $5, $6)`, tableAWSEventsIPSHostnames), timestamp, isPublic, isJoin, resourceID, ipAddress, hostname); err != nil { // nolint
		return err
	}

	return nil
}

// GetIPAddressesForTimeRange gets the audit trail of IP address and hostname changes over a specified time range
func (db *DB) GetIPAddressesForTimeRange(ctx context.Context, start time.Time, end time.Time) ([]domain.NetworkChangeEvent, error) {
	sqlstmt := commonQuery + fmt.Sprintf("%s.ts BETWEEN $1 AND $2;", tableAWSEventsIPSHostnames)
	return db.runQuery(ctx, sqlstmt, start, end)
}

// GetIPAddressesForIPAddress gets the audit trail of IP Address and hostname changes for a specified IP address
func (db *DB) GetIPAddressesForIPAddress(ctx context.Context, ipAddress string) ([]domain.NetworkChangeEvent, error) {
	sqlstmt := commonQuery + fmt.Sprintf("%s.aws_ips_ip = $1;", tableAWSEventsIPSHostnames)
	return db.runQuery(ctx, sqlstmt, ipAddress)
}

// GetIPAddressesForHostname gets the audit trail of IP Address and hostname changes for a specified hostname
func (db *DB) GetIPAddressesForHostname(ctx context.Context, hostname string) ([]domain.NetworkChangeEvent, error) {
	sqlstmt := commonQuery + fmt.Sprintf("%s.aws_hostnames_hostname = $1;", tableAWSEventsIPSHostnames)
	return db.runQuery(ctx, sqlstmt, hostname)
}

func (db *DB) runQuery(ctx context.Context, query string, args ...interface{}) ([]domain.NetworkChangeEvent, error) {
	rows, err := db.sqldb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	networkChangeEvents := make([]domain.NetworkChangeEvent, 0)
	for rows.Next() {
		var row domain.NetworkChangeEvent
		if err = rows.Scan(&row.ResourceID, &row.IPAddress, &row.Hostname, &row.IsPublic, &row.IsJoin, &row.Timestamp, &row.AccountID, &row.Region, &row.Type, &row.Tags); err != nil {
			log.Fatal(err)
			return nil, err
		}
		networkChangeEvents = append(networkChangeEvents, row)
	}

	return networkChangeEvents, nil

}

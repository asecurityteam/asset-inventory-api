package v1

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"

	_ "github.com/lib/pq" // postgres driver for sql must be imported so sql finds and uses it
)

/*
CREATE TABLE aws_resources (id VARCHAR PRIMARY KEY, account_id VARCHAR NOT NULL, region VARCHAR NOT NULL, resource_meta JSONB);
CREATE TABLE aws_ips (ts TIMESTAMP NOT NULL, ip VARCHAR NOT NULL, is_public BOOLEAN NOT NULL, isJoin BOOLEAN NOT NULL, resource_id VARCHAR, FOREIGN KEY (resource_id) REFERENCES aws_resources(id), PRIMARY KEY (ts,ip));
CREATE TABLE aws_hostnames (ts TIMESTAMP NOT NULL, hostname VARCHAR NOT NULL, resource_id VARCHAR, FOREIGN KEY (resource_id) REFERENCES aws_resources(id), PRIMARY KEY (ts,hostname));
CREATE TABLE aws_ips_hostnames (aws_ips_ts TIMESTAMP, aws_ips_ip VARCHAR, aws_hostnames_ts TIMESTAMP, aws_hostnames_hostname VARCHAR, FOREIGN KEY (aws_ips_ts,aws_ips_ip) REFERENCES aws_ips(ts,ip), FOREIGN KEY (aws_hostnames_ts,aws_hostnames_hostname) REFERENCES aws_hostnames(ts,hostname));

A resource appears with private ip 4.3.2.1, and public ip 9.8.7.6, to which google.com and gmail.com are mapped:

INSERT INTO aws_resources VALUES ('a','b','c','{}');
INSERT INTO aws_ips VALUES (make_timestamp(2019, 1, 15, 8, 19, 23.5),'4.3.2.1',false,true,'a');
INSERT INTO aws_ips VALUES (make_timestamp(2019, 1, 15, 8, 19, 23.5),'9.8.7.6',true,true,'a');
INSERT INTO aws_hostnames VALUES (make_timestamp(2019, 1, 15, 8, 19, 23.5),'google.com','a');
INSERT INTO aws_hostnames VALUES (make_timestamp(2019, 1, 15, 8, 19, 23.5),'gmail.com','a');
INSERT INTO aws_ips_hostnames VALUES (make_timestamp(2019, 1, 15, 8, 19, 23.5),'9.8.7.6',make_timestamp(2019, 1, 15, 8, 19, 23.5),'google.com');
INSERT INTO aws_ips_hostnames VALUES (make_timestamp(2019, 1, 15, 8, 19, 23.5),'9.8.7.6',make_timestamp(2019, 1, 15, 8, 19, 23.5),'gmail.com');

Find the ips + hostnames on Jan 15, 2019 between 8:00am and 8:30am.

SELECT
  aws_hostnames.hostname, aws_ips.ip, aws_ips.is_public, aws_ips.isJoin, aws_ips.ts
FROM
	aws_ips
FULL JOIN aws_ips_hostnames
	ON aws_ips.ip = aws_ips_hostnames.aws_ips_ip
	AND aws_ips.ts = aws_ips_hostnames.aws_ips_ts
FULL JOIN aws_hostnames
	ON aws_hostnames.hostname = aws_ips_hostnames.aws_hostnames_hostname
	AND aws_hostnames.ts = aws_ips_hostnames.aws_hostnames_ts
WHERE
	aws_ips.ts BETWEEN
		make_timestamp(2019, 1, 15, 8, 0, 0) AND
		make_timestamp(2019, 1, 15, 8, 30, 0);

  hostname  |   ip    | is_public | isJoin |          ts
------------+---------+-----------+---------+-----------------------
 google.com | 9.8.7.6 | t         | t       | 2019-01-15 08:19:23.5
 gmail.com  | 9.8.7.6 | t         | t       | 2019-01-15 08:19:23.5
            | 4.3.2.1 | f         | t       | 2019-01-15 08:19:23.5
(3 rows)

A minute later, 9.8.7.6 is dropped:

INSERT INTO aws_ips VALUES (make_timestamp(2019, 1, 15, 8, 20, 23.5),'9.8.7.6',true,false,'a');
INSERT INTO aws_hostnames VALUES (make_timestamp(2019, 1, 15, 8, 20, 23.5),'google.com','a');
INSERT INTO aws_hostnames VALUES (make_timestamp(2019, 1, 15, 8, 20, 23.5),'gmail.com','a');
INSERT INTO aws_ips_hostnames VALUES (make_timestamp(2019, 1, 15, 8, 20, 23.5),'9.8.7.6',make_timestamp(2019, 1, 15, 8, 20, 23.5),'google.com');
INSERT INTO aws_ips_hostnames VALUES (make_timestamp(2019, 1, 15, 8, 20, 23.5),'9.8.7.6',make_timestamp(2019, 1, 15, 8, 20, 23.5),'gmail.com');

SELECT shows the drops:

  hostname  |   ip    | is_public | isJoin |          ts
------------+---------+-----------+---------+-----------------------
 google.com | 9.8.7.6 | t         | t       | 2019-01-15 08:19:23.5
 gmail.com  | 9.8.7.6 | t         | t       | 2019-01-15 08:19:23.5
 google.com | 9.8.7.6 | t         | f       | 2019-01-15 08:20:23.5
 gmail.com  | 9.8.7.6 | t         | f       | 2019-01-15 08:20:23.5
			| 4.3.2.1 | f         | t       | 2019-01-15 08:19:23.5
(5 rows)

A minute after that, 9.8.7.6 is taken over by yahoo.com:

INSERT INTO aws_ips VALUES (make_timestamp(2019, 1, 15, 8, 21, 23.5),'9.8.7.6',true,true,'a');
INSERT INTO aws_hostnames VALUES (make_timestamp(2019, 1, 15, 8, 21, 23.5),'yahoo.com','a');
INSERT INTO aws_ips_hostnames VALUES (make_timestamp(2019, 1, 15, 8, 21, 23.5),'9.8.7.6',make_timestamp(2019, 1, 15, 8, 21, 23.5),'yahoo.com');

SELECT shows the join:

  hostname  |   ip    | is_public | isJoin |          ts
------------+---------+-----------+---------+-----------------------
 google.com | 9.8.7.6 | t         | t       | 2019-01-15 08:19:23.5
 gmail.com  | 9.8.7.6 | t         | t       | 2019-01-15 08:19:23.5
 google.com | 9.8.7.6 | t         | f       | 2019-01-15 08:20:23.5
 gmail.com  | 9.8.7.6 | t         | f       | 2019-01-15 08:20:23.5
 yahoo.com  | 9.8.7.6 | t         | t       | 2019-01-15 08:21:23.5
            | 4.3.2.1 | f         | t       | 2019-01-15 08:19:23.5
(6 rows)

*/

// Also, some good advice to follow:  https://www.vividcortex.com/blog/2015/09/22/common-pitfalls-go/

const tableAWSResources = "aws_resources"
const tableAWSIPS = "aws_ips"
const tableAWSHostnames = "aws_hostnames"
const tableAWSIPSHostnames = "aws_ips_hostnames"
const commonQuery = "SELECT " +
	"aws_hostnames.hostname, aws_ips.ip, aws_ips.is_public, aws_ips.isJoin, aws_ips.ts " +
	"FROM " +
	"aws_ips " +
	"FULL JOIN aws_ips_hostnames " +
	"ON aws_ips.ip = aws_ips_hostnames.aws_ips_ip " +
	"AND aws_ips.ts = aws_ips_hostnames.aws_ips_ts " +
	"FULL JOIN aws_hostnames " +
	"ON aws_hostnames.hostname = aws_ips_hostnames.aws_hostnames_hostname " +
	"AND aws_hostnames.ts = aws_ips_hostnames.aws_hostnames_ts " +
	"WHERE "

var once sync.Once

// DB represents a convenient database abstraction layer
type DB struct {
	sqldb *sql.DB      // this is a unit test seam
	LogFn domain.LogFn // also a seam
}

// Init initializes a connection to a Postgres database according to the environment variables POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DATABASE
func (db *DB) Init(ctx context.Context) error {
	var initerr error
	once.Do(func() {

		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")
		dbname := os.Getenv("POSTGRES_DATABASE")

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
	sqlStatement := `INSERT INTO $1 VALUES ($2, $3, $4, $5) ON CONFLICT DO NOTHING RETURNING id`

	// Marshal the map into a JSON string.
	meta, err := json.Marshal(cloudAssetChanges.Tags)
	if err != nil {
		return err
	}

	if _, err := db.sqldb.Exec(sqlStatement, tableAWSResources, cloudAssetChanges.ARN, cloudAssetChanges.AccountID, cloudAssetChanges.Region, string(meta)); err != nil {
		return err
	}

	return nil
}

func (db *DB) recordNetworkChanges(ctx context.Context, resourceID string, timestamp time.Time, changes domain.NetworkChanges, tx *sql.Tx) error {

	for _, hostname := range changes.Hostnames {
		err := db.insertHostname(hostname, resourceID, timestamp, tx)
		if err != nil {
			return err
		}
	}

	for _, val := range changes.PrivateIPAddresses {
		isJoin := false
		if changes.ChangeType == "ADDED" {
			isJoin = true
		}
		err := db.insertIPAddress(val, resourceID, false, isJoin, timestamp, tx)
		if err != nil {
			return err
		}
	}

	for _, val := range changes.PublicIPAddresses {
		isJoin := false
		if changes.ChangeType == "ADDED" {
			isJoin = true
		}
		err := db.insertIPAddress(val, resourceID, true, isJoin, timestamp, tx)
		if err != nil {
			return err
		}
		// yeah, a nested for loop to establish the relationship of every public IP to every hostname
		for _, hostname := range changes.Hostnames {
			err := db.setHostnameToIPAddressRelationship(hostname, val, timestamp, tx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (db *DB) insertIPAddress(ipAddress string, resourceID string, isPublic bool, isJoin bool, timestamp time.Time, tx *sql.Tx) error {
	// this lib won't give back the last INSERTed row ID, so we don't bother with `RETURNING ...`
	// See https://stackoverflow.com/questions/34708509/how-to-use-returning-with-on-conflict-in-postgresql
	if _, err := tx.Exec(`INSERT INTO $1 VALUES ($2, $3, $4, $5, $6)`, tableAWSIPS, timestamp, ipAddress, isPublic, isJoin, resourceID); err != nil {
		return err
	}

	return nil
}

func (db *DB) insertHostname(hostname string, resourceID string, timestamp time.Time, tx *sql.Tx) error {
	// this lib won't give back the last INSERTed row ID, so we don't bother with `RETURNING ...`
	// See https://stackoverflow.com/questions/34708509/how-to-use-returning-with-on-conflict-in-postgresql
	if _, err := tx.Exec(`INSERT INTO $1 VALUES ($2, $3, $4)`, tableAWSHostnames, timestamp, hostname, resourceID); err != nil {
		return err
	}

	return nil
}

func (db *DB) setHostnameToIPAddressRelationship(hostname string, ipAddress string, timestamp time.Time, tx *sql.Tx) error {
	// this lib won't give back the last INSERTed row ID, so we don't bother with `RETURNING ...`
	// See https://stackoverflow.com/questions/34708509/how-to-use-returning-with-on-conflict-in-postgresql
	if _, err := tx.Exec(`INSERT INTO $1 VALUES ($2, $3, $2, $4)`, tableAWSIPSHostnames, timestamp, hostname, ipAddress); err != nil {
		return err
	}

	return nil
}

// GetIPAddressesForTimeRange gets the audit trail of IP address and hostname changes over a specified time range
func (db *DB) GetIPAddressesForTimeRange(ctx context.Context, start time.Time, end time.Time) ([]domain.QueryResult, error) {
	sqlstmt := commonQuery + "aws_ips.ts BETWEEN $1 AND $2;"
	return db.runQuery(ctx, sqlstmt, start, end)
}

// GetIPAddressesForIPAddress gets the audit trail of IP Address and hostname changes for a specified IP address
func (db *DB) GetIPAddressesForIPAddress(ctx context.Context, ipAddress string) ([]domain.QueryResult, error) {
	sqlstmt := commonQuery + "aws_ips.ip = $1;"
	return db.runQuery(ctx, sqlstmt, ipAddress)
}

// GetIPAddressesForHostname gets the audit trail of IP Address and hostname changes for a specified hostname
func (db *DB) GetIPAddressesForHostname(ctx context.Context, hostname string) ([]domain.QueryResult, error) {
	sqlstmt := commonQuery + "aws_hostnames.hostname = $1;"
	return db.runQuery(ctx, sqlstmt, hostname)
}

func (db *DB) runQuery(ctx context.Context, query string, args ...interface{}) ([]domain.QueryResult, error) {
	rows, err := db.sqldb.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	queryResults := make([]domain.QueryResult, 0)
	for rows.Next() {
		var row domain.QueryResult
		var Timestamp string
		if err = rows.Scan(&row.Hostname, &row.IPAddress, &row.IsPublic, &row.IsJoin, &Timestamp); err != nil {
			log.Fatal(err)
			return nil, err
		}
		if row.Timestamp, err = time.Parse(time.RFC3339, Timestamp); err != nil {
			log.Fatal(err)
			return nil, err
		}
		queryResults = append(queryResults, row)
	}

	return queryResults, nil

}

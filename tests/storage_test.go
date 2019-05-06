// +build integration

package inttest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/storage"
	"github.com/asecurityteam/settings"
	packr "github.com/gobuffalo/packr/v2"
	pq "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// db refers to a raw Postgres, without the "storage.DB" abstraction
var db *sql.DB

// dbStorage is the struct with the functions being tested
var dbStorage *storage.DB
var ctx context.Context

func TestMain(m *testing.M) {

	// wipe the database entirely, which will result in testing DB.Init
	// handling of lack of pre-existing database
	sslmode := "disable"
	host := os.Getenv("POSTGRES_HOSTNAME")
	if host != "localhost" && host != "postgres" {
		sslmode = "require"
	}
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=%s",
		host, os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_USERNAME"), os.Getenv("POSTGRES_PASSWORD"), "postgres", sslmode)
	pgdb, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err.Error())
	}
	defer pgdb.Close()

	if err = wipeDatabase(pgdb, os.Getenv("POSTGRES_DATABASENAME")); err != nil {
		panic(err.Error())
	}

	ctx = context.Background()
	source, err := settings.NewEnvSource(os.Environ())
	if err != nil {
		panic(err.Error())
	}

	postgresConfigComponent := &storage.PostgresConfigComponent{}
	dbStorage = new(storage.DB)
	if err = settings.NewComponent(ctx, source, postgresConfigComponent, dbStorage); err != nil {
		panic(err.Error())
	}

	db, err = connectToDB()
	if err != nil {
		panic(err.Error())
	}

	os.Exit(m.Run())
}

func TestNoDBRows(t *testing.T) {

	before(t, dbStorage)

	// code should tolerate no data in the tables

	at, _ := time.Parse(time.RFC3339, "2019-08-08T08:29:35+00:00")
	networkChangeEvents, err := dbStorage.FetchByIP(ctx, at, "2.3.4.5")
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 0, len(networkChangeEvents), "there really should have been zero rows returned")

}

// TestGetStatusByHostnameAtTimestamp1 test that only one asset has the Hostname at the specified timestamp
func TestGetStatusByHostnameAtTimestamp1(t *testing.T) {

	before(t, dbStorage)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"}
	hostnames := []string{"yahoo.com"} // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	hostname := "yahoo.com" // nolint
	at, _ := time.Parse(time.RFC3339, "2019-08-10T08:29:35+00:00")
	networkChangeEvents, err := dbStorage.FetchByHostname(ctx, at, hostname)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp2 test that only one asset has the Hostname at the specified timestamp despite another one using
// the same hostname _after_ the specified timestamp
func TestGetStatusByHostnameAtTimestamp2(t *testing.T) {

	before(t, dbStorage)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"} // nolint
	hostnames := []string{"yahoo.com"}   // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	// just reuse the existing struct
	fakeCloudAssetChange.ARN = "arn2"
	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // August 11
	fakeCloudAssetChange.ChangeTime = timestamp2
	if err := dbStorage.Store(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	hostname := "yahoo.com"                                        // nolint
	at, _ := time.Parse(time.RFC3339, "2019-08-10T08:29:35+00:00") // query is for status on August 10
	networkChangeEvents, err := dbStorage.FetchByHostname(ctx, at, hostname)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp3 test that two assets have the Hostname at the specified timestamp
func TestGetStatusByHostnameAtTimestamp3(t *testing.T) {

	before(t, dbStorage)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"} // nolint
	hostnames := []string{"yahoo.com"}   // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // August 11
	privateIPs2 := []string{"4.3.2.1"}
	publicIPs2 := []string{"8.7.6.5"}
	fakeCloudAssetChange2 := newFakeCloudAssetChange(privateIPs2, publicIPs2, hostnames, timestamp2, `arn2`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange2); err != nil {
		t.Fatal(err.Error())
	}

	hostname := "yahoo.com"                                        // nolint
	at, _ := time.Parse(time.RFC3339, "2019-08-12T08:29:35+00:00") // query is for status on August 12
	networkChangeEvents, err := dbStorage.FetchByHostname(ctx, at, hostname)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 2, len(networkChangeEvents))

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil}, // nolint
		domain.CloudAssetDetails{nil, []string{"8.7.6.5"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn2", nil},    // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp4 test that one asset has the Hostname at the specified timestamp
func TestGetStatusByHostnameAtTimestamp4(t *testing.T) {

	before(t, dbStorage)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"} // nolint
	hostnames := []string{"yahoo.com"}   // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-12T08:29:35+00:00") // August 12
	privateIPs2 := []string{"4.3.2.1"}
	publicIPs2 := []string{"8.7.6.5"}
	fakeCloudAssetChange2 := newFakeCloudAssetChange(privateIPs2, publicIPs2, hostnames, timestamp2, `arn2`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange2); err != nil {
		t.Fatal(err.Error())
	}

	hostname := "yahoo.com"                                        // nolint
	at, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // query is for status on August 11
	networkChangeEvents, err := dbStorage.FetchByHostname(ctx, at, hostname)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp1 test that only one asset has the Hostname at the specified timestamp
func TestGetStatusByIPAddressAtTimestamp1(t *testing.T) {

	before(t, dbStorage)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"} // nolint
	hostnames := []string{"yahoo.com"}   // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	ipAddress := "88.77.66.55" // nolint
	at, _ := time.Parse(time.RFC3339, "2019-08-10T08:29:35+00:00")
	networkChangeEvents, err := dbStorage.FetchByIP(ctx, at, ipAddress)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp2 test that only one asset has the IP address at the specified timestamp despite another one using
// the same IP address _after_ the specified timestamp
func TestGetStatusByIPAddressAtTimestamp2(t *testing.T) {

	before(t, dbStorage)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"}
	hostnames := []string{"yahoo.com"} // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	// just reuse the existing struct
	fakeCloudAssetChange.ARN = "arn2"
	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // August 11
	fakeCloudAssetChange.ChangeTime = timestamp2
	if err := dbStorage.Store(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	ipAddress := "88.77.66.55"
	at, _ := time.Parse(time.RFC3339, "2019-08-10T08:29:35+00:00") // query is for status on August 10
	networkChangeEvents, err := dbStorage.FetchByIP(ctx, at, ipAddress)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp3 test that two assets have the IP address at the specified timestamp
func TestGetStatusByIPAddressAtTimestamp3(t *testing.T) {

	before(t, dbStorage)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"}
	hostnames := []string{"yahoo.com"} // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // August 11
	hostnames2 := []string{"blarg.com"}
	fakeCloudAssetChange2 := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames2, timestamp2, `arn2`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange2); err != nil {
		t.Fatal(err.Error())
	}

	ipAddress := "88.77.66.55"
	at, _ := time.Parse(time.RFC3339, "2019-08-12T08:29:35+00:00") // query is for status on August 12
	networkChangeEvents, err := dbStorage.FetchByIP(ctx, at, ipAddress)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 2, len(networkChangeEvents))

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil},  // nolint
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"blarg.com"}, "rtype", "aid", "region", "arn2", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp4 test that one asset has the IP address at the specified timestamp
func TestGetStatusByIPAddressAtTimestamp4(t *testing.T) {

	before(t, dbStorage)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"}
	hostnames := []string{"yahoo.com"} // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-12T08:29:35+00:00") // August 12
	hostnames2 := []string{"blarg.com"}
	fakeCloudAssetChange2 := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames2, timestamp2, `arn2`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange2); err != nil {
		t.Fatal(err.Error())
	}

	ipAddress := "88.77.66.55"
	at, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // query is for status on August 11
	networkChangeEvents, err := dbStorage.FetchByIP(ctx, at, ipAddress)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp5 test that two assets have the IP address at the specified timestamp, despite another asset
// having then dropping the same IP address prior to that timestamp
func TestGetStatusByIPAddressAtTimestamp5(t *testing.T) {

	before(t, dbStorage)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"}
	hostnames := []string{"yahoo.com"} // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-12T08:29:35+00:00") // August 12
	hostnames2 := []string{"blarg.com"}
	fakeCloudAssetChange2 := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames2, timestamp2, `arn2`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange2); err != nil {
		t.Fatal(err.Error())
	}

	timestamp3, _ := time.Parse(time.RFC3339, "2019-08-10T08:29:35+00:00") // August 10, arn3
	hostnames3 := []string{"reddit.com"}
	fakeCloudAssetChange3 := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames3, timestamp3, `arn3`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange3); err != nil {
		t.Fatal(err.Error())
	}

	timestamp4, _ := time.Parse(time.RFC3339, "2019-08-10T08:39:35+00:00") // August 10, 10 minutes later, arn3 drops the same IP address
	fakeCloudAssetChange4 := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames3, timestamp4, `arn3`, `rtype`, `aid`, `region`, nil, false)
	if err := dbStorage.Store(ctx, fakeCloudAssetChange4); err != nil {
		t.Fatal(err.Error())
	}

	ipAddress := "88.77.66.55"
	at, _ := time.Parse(time.RFC3339, "2019-08-13T08:29:35+00:00") // query is for status on August 13
	networkChangeEvents, err := dbStorage.FetchByIP(ctx, at, ipAddress)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 2, len(networkChangeEvents))

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil},  // nolint
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"blarg.com"}, "rtype", "aid", "region", "arn2", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

func TestGeneratePartitions(t *testing.T) {
	before(t, dbStorage) // start with a parition from 07/2019-09/2019
	partitions := getPartitions(t)
	require.Equal(t, 1, len(partitions))
	require.Equal(t, "aws_events_ips_hostnames_2019_08to2019_11", partitions[0])

	err := dbStorage.GeneratePartition(context.Background())
	require.NoError(t, err)
	partitions = getPartitions(t)
	require.Equal(t, 2, len(partitions))
	require.Equal(t, "aws_events_ips_hostnames_2019_11to2020_02", partitions[0])

	err = dbStorage.GeneratePartition(context.Background())
	require.NoError(t, err)
	partitions = getPartitions(t)
	require.Equal(t, 3, len(partitions))
	require.Equal(t, "aws_events_ips_hostnames_2020_02to2020_05", partitions[0])

	// conflict
	err = dbStorage.GeneratePartitionWithTimestamp(context.Background(), time.Date(2020, 02, 01, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
	_, ok := err.(domain.PartitionConflict)
	require.True(t, ok, fmt.Sprintf("Expected PartitionConflict, but received %t", err))

	// conflict
	err = dbStorage.GeneratePartitionWithTimestamp(context.Background(), time.Date(2020, 03, 01, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
	_, ok = err.(domain.PartitionConflict)
	require.True(t, ok, fmt.Sprintf("Expected PartitionConflict, but received %t", err))

	// conflict
	err = dbStorage.GeneratePartitionWithTimestamp(context.Background(), time.Date(2020, 04, 01, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
	_, ok = err.(domain.PartitionConflict)
	require.True(t, ok, fmt.Sprintf("Expected PartitionConflict, but received %t", err))
}

func getPartitions(t *testing.T) []string {
	rows, err := db.Query(`SELECT name FROM partitions ORDER BY partition_begin DESC`)
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		t.Fatalf("Received error checking partitions: %v", err)
	}
	partitions := make([]string, 0)
	for rows.Next() {
		var partition string
		if err := rows.Scan(&partition); err != nil {
			t.Fatalf("Received error scanning next partition: %v", err)
		}
		partitions = append(partitions, partition)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("Received error traversing partitions: %v", err)
	}
	return partitions
}

// returns a raw sql.DB object, rather than the storage.DB abstraction, so
// we can perform some Postgres cleanup/prep/checks that are test-specific
func connectToDB() (*sql.DB, error) {
	host := os.Getenv("POSTGRES_HOSTNAME")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USERNAME")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DATABASENAME")

	sslmode := "disable"
	if host != "localhost" && host != "postgres" {
		sslmode = "require"
	}
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
	pgdb, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	err = pgdb.Ping()
	if err != nil {
		return nil, err
	}

	return pgdb, nil
}

// before is the function all tests should call to ensure no state is carried over
// from prior tests
func before(t *testing.T, db *storage.DB) {
	require.NoError(t, db.RunScript(context.Background(), "1_clean.sql"))
	require.NoError(t, db.RunScript(context.Background(), "2_create.sql"))
	require.NoError(t, db.GeneratePartitionWithTimestamp(context.Background(), time.Date(2019, time.August, 1, 0, 0, 0, 0, time.UTC)))
}

// dropTables is a utility function called by "before"
func wipeDatabase(db *sql.DB, dbName string) error {

	sqlFile := "0_wipe.sql"

	box := packr.New("box", "../scripts")
	_, err := box.Find(sqlFile)
	if err != nil {
		return err
	}
	s, err := box.FindString(sqlFile)
	if err != nil {
		return err
	}

	if _, err = db.Exec(fmt.Sprintf(s, dbName)); err != nil {
		if driverErr, ok := err.(*pq.Error); ok {
			if strings.EqualFold(driverErr.Code.Name(), "invalid_catalog_name") { // from https://www.postgresql.org/docs/11/errcodes-appendix.html
				// it's ok the DB does not exist; this might by the very first run
				return nil
			}
		}
		return err
	}

	return nil
}

// newFakeCloudAssetChange is a utility function to create the struct that is the inbound change report we need to save
func newFakeCloudAssetChange(privateIPs []string, publicIPs []string, hostnames []string, timestamp time.Time, arn string, resourceType string, accountID string, region string, tags map[string]string, added bool) domain.CloudAssetChanges { // nolint
	eventType := "ADDED"
	if !added {
		eventType = "DELETED"
	}
	networkChangesArray := []domain.NetworkChanges{
		domain.NetworkChanges{
			PrivateIPAddresses: privateIPs,
			PublicIPAddresses:  publicIPs,
			Hostnames:          hostnames,
			ChangeType:         eventType,
		},
	}
	cloudAssetChanges := domain.CloudAssetChanges{
		Changes:      networkChangesArray,
		ChangeTime:   timestamp,
		ResourceType: resourceType,
		AccountID:    accountID,
		Region:       region,
		ARN:          arn,
		Tags:         tags,
	}

	return cloudAssetChanges
}

func assertArrayEqualIgnoreOrder(t *testing.T, expected, actual []domain.CloudAssetDetails) {
	// brute force
	assert.Equal(t, len(expected), len(actual))
	equalityCount := 0
	for _, expectedVal := range expected {
		for _, actualVal := range actual {

			e, _ := json.Marshal(expectedVal)
			a, _ := json.Marshal(actualVal)

			// likely due to timestamp, DeepEqual(expectedVal, actualVal) would not work, so checking the Marshaled JSON:
			if reflect.DeepEqual(e, a) {
				equalityCount++
				break
			}
		}
	}
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)
	assert.Equalf(t, len(expected), equalityCount, "Expected results differ from actual.  Expected: %s  Actual: %s", string(expectedJSON), string(actualJSON))
}

// +build integration

package inttest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/storage"
	"github.com/asecurityteam/logevent"
	"github.com/asecurityteam/runhttp"
	"github.com/asecurityteam/settings"
	packr "github.com/gobuffalo/packr/v2"
	"github.com/stretchr/testify/assert"
)

// db refers to a raw Postgres, without the "storage.DB" abstraction
var db *sql.DB

// dbStorage is the struct with the functions being tested
var dbStorage *storage.DB
var ctx context.Context

func TestMain(m *testing.M) {
	ctx = context.Background()
	ctx = logevent.NewContext(ctx, stdoutLogger())
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

	before(t, db)

	// code should tolerate no data in the tables

	from, _ := time.Parse(time.RFC3339, "2019-08-08T08:29:35+00:00")
	to, _ := time.Parse(time.RFC3339, "2019-08-10T08:29:35+00:00")
	networkChangeEvents, err := dbStorage.GetIPAddressesForTimeRange(ctx, from, to)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 0, len(networkChangeEvents), "there really should have been zero rows returned")

}

func TestGetIPAddressesForTimeRange(t *testing.T) {

	before(t, db)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"} // nolint
	hostnames := []string{"yahoo.com"}   // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	from, _ := time.Parse(time.RFC3339, "2019-08-08T08:29:35+00:00")
	to, _ := time.Parse(time.RFC3339, "2019-08-10T08:29:35+00:00")
	networkChangeEvents, err := dbStorage.GetIPAddressesForTimeRange(ctx, from, to)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 2, len(networkChangeEvents))

	expected := []domain.NetworkChangeEvent{
		domain.NetworkChangeEvent{"arn", "88.77.66.55", "yahoo.com", true, true, timestamp, "aid", "region", "rtype", nil}, // nolint
		domain.NetworkChangeEvent{"arn", "44.33.22.11", "", false, true, timestamp, "aid", "region", "rtype", nil},
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

func TestGetIPAddressesForHostname(t *testing.T) {

	before(t, db)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"} // nolint
	hostnames := []string{"yahoo.com"}   // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	networkChangeEvents, err := dbStorage.GetIPAddressesForHostname(ctx, "yahoo.com") // nolint
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.NetworkChangeEvent{
		domain.NetworkChangeEvent{"arn", "88.77.66.55", "yahoo.com", true, true, timestamp, "aid", "region", "rtype", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

func TestGetIPAddressesForIPAddress(t *testing.T) {

	before(t, db)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"} // nolint
	hostnames := []string{"yahoo.com"}   // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	networkChangeEvents, err := dbStorage.GetIPAddressesForIPAddress(ctx, "44.33.22.11")
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.NetworkChangeEvent{
		domain.NetworkChangeEvent{"arn", "44.33.22.11", "", false, true, timestamp, "aid", "region", "rtype", nil},
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp1 test that only one asset has the Hostname at the specified timestamp
func TestGetStatusByHostnameAtTimestamp1(t *testing.T) {

	before(t, db)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"}
	hostnames := []string{"yahoo.com"} // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	hostname := "yahoo.com" // nolint
	at, _ := time.Parse(time.RFC3339, "2019-08-10T08:29:35+00:00")
	networkChangeEvents, err := dbStorage.FetchByHostname(ctx, at, hostname)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.NetworkChangeEvent{
		domain.NetworkChangeEvent{"arn", "88.77.66.55", "yahoo.com", true, true, timestamp, "aid", "region", "rtype", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp2 test that only one asset has the Hostname at the specified timestamp despite another one using
// the same hostname _after_ the specified timestamp
func TestGetStatusByHostnameAtTimestamp2(t *testing.T) {

	before(t, db)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"} // nolint
	hostnames := []string{"yahoo.com"}   // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	// just reuse the existing struct
	fakeCloudAssetChange.ARN = "arn2"
	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // August 11
	fakeCloudAssetChange.ChangeTime = timestamp2
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	hostname := "yahoo.com"                                        // nolint
	at, _ := time.Parse(time.RFC3339, "2019-08-10T08:29:35+00:00") // query is for status on August 10
	networkChangeEvents, err := dbStorage.FetchByHostname(ctx, at, hostname)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.NetworkChangeEvent{
		domain.NetworkChangeEvent{"arn", "88.77.66.55", "yahoo.com", true, true, timestamp, "aid", "region", "rtype", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp3 test that two assets have the Hostname at the specified timestamp
func TestGetStatusByHostnameAtTimestamp3(t *testing.T) {

	before(t, db)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"} // nolint
	hostnames := []string{"yahoo.com"}   // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // August 11
	privateIPs2 := []string{"4.3.2.1"}
	publicIPs2 := []string{"8.7.6.5"}
	fakeCloudAssetChange2 := newFakeCloudAssetChange(privateIPs2, publicIPs2, hostnames, timestamp2, `arn2`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange2); err != nil {
		t.Fatal(err.Error())
	}

	hostname := "yahoo.com"                                        // nolint
	at, _ := time.Parse(time.RFC3339, "2019-08-12T08:29:35+00:00") // query is for status on August 12
	networkChangeEvents, err := dbStorage.FetchByHostname(ctx, at, hostname)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 2, len(networkChangeEvents))

	expected := []domain.NetworkChangeEvent{
		domain.NetworkChangeEvent{"arn", "88.77.66.55", "yahoo.com", true, true, timestamp, "aid", "region", "rtype", nil}, // nolint
		domain.NetworkChangeEvent{"arn2", "8.7.6.5", "yahoo.com", true, true, timestamp2, "aid", "region", "rtype", nil},   // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp4 test that one asset has the Hostname at the specified timestamp
func TestGetStatusByHostnameAtTimestamp4(t *testing.T) {

	before(t, db)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"} // nolint
	hostnames := []string{"yahoo.com"}   // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-12T08:29:35+00:00") // August 12
	privateIPs2 := []string{"4.3.2.1"}
	publicIPs2 := []string{"8.7.6.5"}
	fakeCloudAssetChange2 := newFakeCloudAssetChange(privateIPs2, publicIPs2, hostnames, timestamp2, `arn2`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange2); err != nil {
		t.Fatal(err.Error())
	}

	hostname := "yahoo.com"                                        // nolint
	at, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // query is for status on August 11
	networkChangeEvents, err := dbStorage.FetchByHostname(ctx, at, hostname)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.NetworkChangeEvent{
		domain.NetworkChangeEvent{"arn", "88.77.66.55", "yahoo.com", true, true, timestamp, "aid", "region", "rtype", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp1 test that only one asset has the Hostname at the specified timestamp
func TestGetStatusByIPAddressAtTimestamp1(t *testing.T) {

	before(t, db)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"} // nolint
	hostnames := []string{"yahoo.com"}   // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	ipAddress := "88.77.66.55" // nolint
	at, _ := time.Parse(time.RFC3339, "2019-08-10T08:29:35+00:00")
	networkChangeEvents, err := dbStorage.FetchByIP(ctx, at, ipAddress)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.NetworkChangeEvent{
		domain.NetworkChangeEvent{"arn", "88.77.66.55", "yahoo.com", true, true, timestamp, "aid", "region", "rtype", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp2 test that only one asset has the Hostname at the specified timestamp despite another one using
// the same hostname _after_ the specified timestamp
func TestGetStatusByIPAddressAtTimestamp2(t *testing.T) {

	before(t, db)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"}
	hostnames := []string{"yahoo.com"} // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	// just reuse the existing struct
	fakeCloudAssetChange.ARN = "arn2"
	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // August 11
	fakeCloudAssetChange.ChangeTime = timestamp2
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	ipAddress := "88.77.66.55"
	at, _ := time.Parse(time.RFC3339, "2019-08-10T08:29:35+00:00") // query is for status on August 10
	networkChangeEvents, err := dbStorage.FetchByIP(ctx, at, ipAddress)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.NetworkChangeEvent{
		domain.NetworkChangeEvent{"arn", "88.77.66.55", "yahoo.com", true, true, timestamp, "aid", "region", "rtype", nil},
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp3 test that two assets have the Hostname at the specified timestamp
func TestGetStatusByIPAddressAtTimestamp3(t *testing.T) {

	before(t, db)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"}
	hostnames := []string{"yahoo.com"} // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // August 11
	hostnames2 := []string{"blarg.com"}
	fakeCloudAssetChange2 := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames2, timestamp2, `arn2`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange2); err != nil {
		t.Fatal(err.Error())
	}

	ipAddress := "88.77.66.55"
	at, _ := time.Parse(time.RFC3339, "2019-08-12T08:29:35+00:00") // query is for status on August 12
	networkChangeEvents, err := dbStorage.FetchByIP(ctx, at, ipAddress)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 2, len(networkChangeEvents))

	expected := []domain.NetworkChangeEvent{
		domain.NetworkChangeEvent{"arn", "88.77.66.55", "yahoo.com", true, true, timestamp, "aid", "region", "rtype", nil},
		domain.NetworkChangeEvent{"arn2", "88.77.66.55", "blarg.com", true, true, timestamp2, "aid", "region", "rtype", nil},
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp4 test that one asset has the Hostname at the specified timestamp
func TestGetStatusByIPAddressAtTimestamp4(t *testing.T) {

	before(t, db)

	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"}
	hostnames := []string{"yahoo.com"} // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange); err != nil {
		t.Fatal(err.Error())
	}

	timestamp2, _ := time.Parse(time.RFC3339, "2019-08-12T08:29:35+00:00") // August 12
	hostnames2 := []string{"blarg.com"}
	fakeCloudAssetChange2 := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames2, timestamp2, `arn2`, `rid`, `rtype`, `aid`, `region`, nil, true)
	if err := dbStorage.StoreCloudAsset(ctx, fakeCloudAssetChange2); err != nil {
		t.Fatal(err.Error())
	}

	ipAddress := "88.77.66.55"
	at, _ := time.Parse(time.RFC3339, "2019-08-11T08:29:35+00:00") // query is for status on August 11
	networkChangeEvents, err := dbStorage.FetchByIP(ctx, at, ipAddress)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, 1, len(networkChangeEvents))

	expected := []domain.NetworkChangeEvent{
		domain.NetworkChangeEvent{"arn", "88.77.66.55", "yahoo.com", true, true, timestamp, "aid", "region", "rtype", nil}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

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
func before(t *testing.T, db *sql.DB) {
	if err := dropTables(db); err != nil {
		t.Fatalf("Failed to DROP tables due to: %s", err.Error())
	}
	if err := createTables(db); err != nil {
		t.Fatalf("Failed to CREATE tables due to: %s", err.Error())
	}
}

// dropTables is a utility function called by "before"
func dropTables(db *sql.DB) error {

	sqlFile := "1_clean.sql"

	box := packr.New("box", "../scripts")
	path, err := box.Find(sqlFile)
	if err != nil {
		return err
	}
	s, err := box.FindString(sqlFile)
	if err != nil {
		return err
	}

	fmt.Printf("DROPping existing aws_* tables %s\n", string(path))

	if _, err = db.Exec(s); err != nil {
		return err
	}

	return nil
}

// createTables is a utility function called by "before"
func createTables(db *sql.DB) error {

	sqlFile := "2_create.sql"

	box := packr.New("box", "../scripts")
	path, err := box.Find(sqlFile)
	if err != nil {
		return err
	}
	s, err := box.FindString(sqlFile)
	if err != nil {
		return err
	}

	fmt.Printf("CREATEing aws_* tables using %s\n", string(path))

	if _, err = db.Exec(s); err != nil {
		return err
	}

	return nil

}

// newFakeCloudAssetChange is a utility function to create the struct that is the inbound change report we need to save
func newFakeCloudAssetChange(privateIPs []string, publicIPs []string, hostnames []string, timestamp time.Time, arn string, resourceID string, resourceType string, accountID string, region string, tags map[string]string, added bool) domain.CloudAssetChanges { // nolint
	eventType := "ADDED"
	if !added {
		eventType = "DELETED"
	}
	networkChangesArray := []domain.NetworkChanges{domain.NetworkChanges{privateIPs, publicIPs, hostnames, eventType}}
	cloudAssetChanges := domain.CloudAssetChanges{networkChangesArray, timestamp, resourceType, accountID, region, resourceID, arn, tags}

	return cloudAssetChanges
}

func assertArrayEqualIgnoreOrder(t *testing.T, expected, actual []domain.NetworkChangeEvent) {
	// brute force
	assert.Equal(t, len(expected), len(actual))
	equalityCount := 0
	for _, expectedVal := range expected {
		for _, actualVal := range actual {

			e, _ := json.Marshal(expectedVal)
			a, _ := json.Marshal(actualVal)

			// likely due to timestamp, DeepEqual(expectedVal, actualVal) would not work, so checking the Marhalled JSON:
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

// stdoutLogger is a logger that prints to stdout, Captain Obvious
func stdoutLogger() runhttp.Logger {
	return logevent.New(logevent.Config{Output: os.Stdout})
}

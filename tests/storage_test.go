// +build integration

package inttest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/storage"
)

// dbStorage is the struct with the functions being tested
var dbStorage *storage.DB
var dbSchema *storage.SchemaManager
var ctx context.Context

// possibly not the best way to parametrise tests, but it works
var testWithSchemaVersion = storage.MinimumSchemaVersion

func TestMain(m *testing.M) {
	ctx = context.Background()
	pgComponent := storage.PostgresConfigComponent{}
	pgSettings := pgComponent.Settings()
	pgSettings.URL = os.Getenv("POSTGRES_URL")
	var err error
	dbStorage, err = pgComponent.New(ctx, pgSettings, storage.Primary)
	if err != nil {
		panic(err)
	}
	dbSchema, err = storage.NewSchemaManager(pgSettings.MigrationsPath, pgSettings.URL)
	if err != nil {
		panic(err)
	}
	// none of the known DB versions after initial should EVER result in any of the tests failing, so we test all of them
	suitResult := 0
	for ver := storage.MinimumSchemaVersion; ver <= storage.NewSchemaOnlyVersion; ver++ {
		testWithSchemaVersion = ver
		res := m.Run()
		if res != 0 {
			suitResult = res
		}
	}
	os.Exit(suitResult) //non-zero return value if any of the runs failed
}

func TestNoDBRows(t *testing.T) {

	before(t, dbStorage, dbSchema)

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

	before(t, dbStorage, dbSchema)

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
		domain.CloudAssetDetails{ //nolint
			nil,
			[]string{"88.77.66.55"},
			[]string{"yahoo.com"},
			"rtype",
			"aid",
			"region",
			"arn",
			nil,
			domain.AccountOwner{},
		},
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp2 test that only one asset has the Hostname at the specified timestamp despite another one using
// the same hostname _after_ the specified timestamp
func TestGetStatusByHostnameAtTimestamp2(t *testing.T) {

	before(t, dbStorage, dbSchema)

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

	expected := []domain.CloudAssetDetails{{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil, domain.AccountOwner{}}} // nolint

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp3 test that two assets have the Hostname at the specified timestamp
func TestGetStatusByHostnameAtTimestamp3(t *testing.T) {

	before(t, dbStorage, dbSchema)

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
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil, domain.AccountOwner{}}, // nolint
		domain.CloudAssetDetails{nil, []string{"8.7.6.5"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn2", nil, domain.AccountOwner{}},    // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByHostnameAtTimestamp4 test that one asset has the Hostname at the specified timestamp
func TestGetStatusByHostnameAtTimestamp4(t *testing.T) {

	before(t, dbStorage, dbSchema)

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
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil, domain.AccountOwner{}}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp1 test that only one asset has the IP address at the specified timestamp
func TestGetStatusByIPAddressAtTimestamp1(t *testing.T) {

	before(t, dbStorage, dbSchema)

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
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil, domain.AccountOwner{}}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp2 test that only one asset has the IP address at the specified timestamp despite another one using
// the same IP address _after_ the specified timestamp
func TestGetStatusByIPAddressAtTimestamp2(t *testing.T) {

	before(t, dbStorage, dbSchema)

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
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil, domain.AccountOwner{}}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp3 test that two assets have the IP address at the specified timestamp
func TestGetStatusByIPAddressAtTimestamp3(t *testing.T) {

	before(t, dbStorage, dbSchema)

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
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil, domain.AccountOwner{}},  // nolint
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"blarg.com"}, "rtype", "aid", "region", "arn2", nil, domain.AccountOwner{}}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp4 test that one asset has the IP address at the specified timestamp
func TestGetStatusByIPAddressAtTimestamp4(t *testing.T) {

	before(t, dbStorage, dbSchema)

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
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil, domain.AccountOwner{}}, // nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

// TestGetStatusByIPAddressAtTimestamp5 test that two assets have the IP address at the specified timestamp, despite another asset
// having then dropping the same IP address prior to that timestamp
func TestGetStatusByIPAddressAtTimestamp5(t *testing.T) {

	before(t, dbStorage, dbSchema)

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
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"yahoo.com"}, "rtype", "aid", "region", "arn", nil, domain.AccountOwner{}},  //nolint
		domain.CloudAssetDetails{nil, []string{"88.77.66.55"}, []string{"blarg.com"}, "rtype", "aid", "region", "arn2", nil, domain.AccountOwner{}}, //nolint
	}

	assertArrayEqualIgnoreOrder(t, expected, networkChangeEvents)

}

func TestGeneratePartitions(t *testing.T) {
	before(t, dbStorage, dbSchema) // start with a partition from 07/2019-09/2019
	partitions, err := dbStorage.GetPartitions(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, len(partitions))
	require.Equal(t, "aws_events_ips_hostnames_2019_08_01to2019_10_30", partitions[0].Name)
	require.True(t, time.Date(2019, time.August, 01, 0, 0, 0, 0, time.UTC).Equal(partitions[0].Begin), fmt.Sprintf("Expected %v to be 2019-08-01T00:00:00Z", partitions[0].Begin))
	require.True(t, time.Date(2019, time.October, 30, 0, 0, 0, 0, time.UTC).Equal(partitions[0].End), fmt.Sprintf("Expected %v to be 2019-10-30T00:00:00Z", partitions[0].End))
	// conflict
	err = dbStorage.GeneratePartition(context.Background(), time.Date(2019, 07, 01, 0, 0, 0, 0, time.UTC), 0)
	require.Error(t, err)
	_, ok := err.(domain.PartitionConflict)
	require.True(t, ok, fmt.Sprintf("Expected PartitionConflict, but received %t", err))

	// create partition before
	err = dbStorage.GeneratePartition(context.Background(), time.Date(2019, 07, 01, 0, 0, 0, 0, time.UTC), 31)
	require.NoError(t, err)
	partitions, err = dbStorage.GetPartitions(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, len(partitions))
	require.Equal(t, "aws_events_ips_hostnames_2019_07_01to2019_08_01", partitions[1].Name)
	require.True(t, time.Date(2019, time.July, 01, 0, 0, 0, 0, time.UTC).Equal(partitions[1].Begin), fmt.Sprintf("Expected %v to be 2019-07-01T00:00:00Z", partitions[1].Begin))
	require.True(t, time.Date(2019, time.August, 01, 0, 0, 0, 0, time.UTC).Equal(partitions[1].End), fmt.Sprintf("Expected %v to be 2019-08-01T00:00:00Z", partitions[1].End))
}

func TestPartitionCounts(t *testing.T) {
	before(t, dbStorage, dbSchema)
	if testWithSchemaVersion >= storage.NewSchemaOnlyVersion { // no partition use with writes only to new schema
		return
	}
	privateIPs := []string{"44.33.22.11"}
	publicIPs := []string{"88.77.66.55"}
	hostnames := []string{"yahoo.com"} // nolint
	timestamp, _ := time.Parse(time.RFC3339, "2019-08-09T08:29:35+00:00")

	fakeCloudAssetChange := newFakeCloudAssetChange(privateIPs, publicIPs, hostnames, timestamp, `arn`, `rtype`, `aid`, `region`, nil, true)
	err := dbStorage.Store(context.Background(), fakeCloudAssetChange)
	require.NoError(t, err, "Error storing asset")

	partitions, err := dbStorage.GetPartitions(context.Background())
	require.NoError(t, err, "Error fetching partitions")
	require.Equal(t, 1, len(partitions))
	require.Equal(t, 2, partitions[0].Count) //IP addresses are inserted into separate rows

}

func TestDeletePartitions(t *testing.T) {
	before(t, dbStorage, dbSchema) // start with a partition from 07/2019-09/2019

	ts := time.Now().Truncate(24 * time.Hour).UTC()
	maxAge := 365
	duration := 14
	numPartitions := 4
	ts = ts.AddDate(0, 0, -maxAge)
	for i := 0; i < numPartitions; i++ {
		ts = ts.AddDate(0, 0, -duration)
		err := dbStorage.GeneratePartition(context.Background(), ts, duration)
		require.NoError(t, err)
	}

	partitions, err := dbStorage.GetPartitions(context.Background())
	require.NoError(t, err)
	require.Equal(t, numPartitions+1, len(partitions))

	name := partitions[0].Name
	err = dbStorage.DeletePartitions(context.Background(), name)
	require.NoError(t, err)

	partitions, err = dbStorage.GetPartitions(context.Background())
	require.NoError(t, err)
	partitionNames := make([]string, 0, len(partitions))
	for _, part := range partitions {
		partitionNames = append(partitionNames, part.Name)
	}
	assert.NotContains(t, partitionNames, name)
}

func TestDeleteNotFoundPartition(t *testing.T) {
	before(t, dbStorage, dbSchema)

	ts := time.Now().Truncate(24 * time.Hour).UTC()
	maxAge := 365
	duration := 14
	numPartitions := 4
	ts = ts.AddDate(0, 0, -maxAge)
	for i := 0; i < numPartitions; i++ {
		ts = ts.AddDate(0, 0, -duration)
		err := dbStorage.GeneratePartition(context.Background(), ts, duration)
		require.NoError(t, err)
	}

	partitions, err := dbStorage.GetPartitions(context.Background())
	require.NoError(t, err)
	require.Equal(t, numPartitions+1, len(partitions))

	partitionNames := make([]string, 0, len(partitions))
	for _, part := range partitions {
		partitionNames = append(partitionNames, part.Name)
	}
	nonexistentPartition := partitions[0].Name + "_notFoundTest"
	require.NotContains(t, partitionNames, nonexistentPartition)

	err = dbStorage.DeletePartitions(context.Background(), nonexistentPartition)
	require.Error(t, err)
	_, ok := err.(domain.NotFoundPartition)
	assert.True(t, ok)

	partitions, err = dbStorage.GetPartitions(context.Background())
	require.NoError(t, err)
	partitionNames = make([]string, 0, len(partitions))
	for _, part := range partitions {
		partitionNames = append(partitionNames, part.Name)
	}

	assert.Equal(t, numPartitions+1, len(partitionNames))
}

// before is the function all tests should call to ensure no state is carried over
// from prior tests
func before(t *testing.T, db *storage.DB, sm *storage.SchemaManager) {
	v, dirty, err := sm.GetSchemaVersion(context.Background())
	if dirty {
		t.Fatalf("schema is marked dirty, refusing to proceed")
	}
	if err != nil { //the migrations mechanism was not initialized yet
		require.NoError(t, sm.MigrateSchemaToVersion(context.Background(), testWithSchemaVersion))
		v = testWithSchemaVersion
	}
	// wipe the database
	for version := v; version > storage.EmptySchemaVersion; {
		version, err = sm.MigrateSchemaDown(context.Background())
		assert.NoError(t, err)
	}
	// re-create the tables with supported schema
	assert.NoError(t, sm.MigrateSchemaToVersion(context.Background(), testWithSchemaVersion))
	assert.NoError(t, db.GeneratePartition(context.Background(), time.Date(2019, time.August, 1, 0, 0, 0, 0, time.UTC), 0))
}

// newFakeCloudAssetChange is a utility function to create the struct that is the inbound change report we need to save
func newFakeCloudAssetChange(privateIPs []string, publicIPs []string, hostnames []string, timestamp time.Time, arn string, resourceType string, accountID string, region string, tags map[string]string, added bool) domain.CloudAssetChanges { // nolint
	eventType := "ADDED"
	if !added {
		eventType = "DELETED"
	}
	networkChangesArray := []domain.NetworkChanges{
		{
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

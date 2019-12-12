package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var scriptText = "SELECT 1"
var scriptNotFound = func(string) (string, error) { return "", errors.New("not found") }
var scriptFound = func(string) (string, error) { return scriptText, nil }

func TestDBInitHandleOpenError(t *testing.T) {
	thedb := DB{}

	hostname := "this is not a hostname"
	port := uint16(99)
	username := "me!"
	password := "mypassword!"
	databasename := "name"
	partitionTTL := 2

	if err := thedb.Init(context.Background(), hostname, port, username, password, databasename, partitionTTL); err == nil {
		t.Errorf("DB.Init should have returned a non-nil error")
	}
}

func TestDoesDBExistTrue(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery("SELECT datname FROM pg_catalog.pg_database WHERE").WithArgs("somename").WillReturnRows(rows).RowsWillBeClosed()

	exists, _ := thedb.doesDBExist("somename")
	if !exists {
		t.Errorf("DB.doesDBExist should have returned true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestDoesDBExistFalse(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	mock.ExpectQuery("SELECT datname FROM pg_catalog.pg_database WHERE").WithArgs("somename").WillReturnError(sql.ErrNoRows)

	exists, _ := thedb.doesDBExist("somename")
	if exists {
		t.Errorf("DB.doesDBExist should have returned false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestDoesDBExistError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	mock.ExpectQuery("SELECT datname FROM pg_catalog.pg_database WHERE").WithArgs("somename").WillReturnError(errors.New("unexpected error"))

	_, err = thedb.doesDBExist("somename")
	if err == nil {
		t.Errorf("DB.doesDBExist should have returned a non-nil error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCreateDBSuccess(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	mock.ExpectExec("CREATE DATABASE").WillReturnResult(sqlmock.NewResult(1, 1))

	err = thedb.create("somename")
	if err != nil {
		t.Errorf("DB.create should have returned a nil error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCreateDBError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	mock.ExpectExec("CREATE DATABASE").WillReturnError(errors.New("unexpected error"))

	err = thedb.create("somename")
	if err == nil {
		t.Errorf("DB.create should have returned a non-nil error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

}

func TestDBUseError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	mock.ExpectClose().WillReturnError(errors.New("unexpected error"))

	err = thedb.use("somename")
	if err == nil {
		t.Errorf("DB.use should have returned a non-nil error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

}

func TestGracefulHandlingOfTxBeginFailure(t *testing.T) {
	// no panics, in other words
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptNotFound,
	}

	mock.ExpectBegin().WillReturnError(fmt.Errorf("could not start transaction"))

	ctx := context.Background()

	if err = thedb.Store(ctx, fakeCloudAssetChanges()); err == nil {
		t.Errorf("was expecting an error, but there was none")
	}
	assert.Equal(t, "could not start transaction", err.Error())

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestShouldINSERTResource(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptNotFound,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("arn", "aid", "region", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnResult(sqlmock.NewResult(1, 1))

	fakeContext, _ := mockdb.BeginTx(context.Background(), nil)

	ctx := context.Background()

	if err = thedb.saveResource(ctx, fakeCloudAssetChanges(), fakeContext); err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestShouldRollbackOnFailureToINSERT(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptNotFound,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("arn", "aid", "region", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnError(fmt.Errorf("some error"))
	mock.ExpectRollback()

	ctx := context.Background()

	if err = thedb.Store(ctx, fakeCloudAssetChanges()); err == nil {
		t.Errorf("was expecting an error, but there was none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestShouldRollbackOnFailureToINSERT2(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptNotFound,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("arn", "aid", "region", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnResult(sqlmock.NewResult(1, 1))
	timestamp, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	mock.ExpectExec("INSERT INTO " + tableAWSHostnames).WithArgs("google.com").WillReturnResult(sqlmock.NewResult(1, 1))                                         // nolint
	mock.ExpectExec("INSERT INTO " + tableAWSIPS).WithArgs("4.3.2.1").WillReturnResult(sqlmock.NewResult(1, 1))                                                  // nolint
	mock.ExpectExec("INSERT INTO "+tableAWSEventsIPSHostnames).WithArgs(timestamp, false, true, "arn", "4.3.2.1", nil).WillReturnError(fmt.Errorf("some error")) // nolint
	mock.ExpectRollback()

	ctx := context.Background()

	if err = thedb.Store(ctx, fakeCloudAssetChanges()); err == nil {
		t.Errorf("was expecting an error, but there was none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGoldenPath(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptNotFound,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("arn", "aid", "region", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnResult(sqlmock.NewResult(1, 1))
	timestamp, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	mock.ExpectExec("INSERT INTO " + tableAWSHostnames).WithArgs("google.com").WillReturnResult(sqlmock.NewResult(1, 1))                                                 // nolint
	mock.ExpectExec("INSERT INTO " + tableAWSIPS).WithArgs("4.3.2.1").WillReturnResult(sqlmock.NewResult(1, 1))                                                          // nolint
	mock.ExpectExec("INSERT INTO "+tableAWSEventsIPSHostnames).WithArgs(timestamp, false, true, "arn", "4.3.2.1", nil).WillReturnResult(sqlmock.NewResult(1, 1))         // nolint
	mock.ExpectExec("INSERT INTO " + tableAWSIPS).WithArgs("8.7.6.5").WillReturnResult(sqlmock.NewResult(1, 1))                                                          // nolint
	mock.ExpectExec("INSERT INTO "+tableAWSEventsIPSHostnames).WithArgs(timestamp, true, true, "arn", "8.7.6.5", "google.com").WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectCommit()

	ctx := context.Background()

	if err = thedb.Store(ctx, fakeCloudAssetChanges()); err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetIPsAtTime(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptNotFound,
	}

	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	ipAddress := "9.8.7.6"
	rowts := "2019-01-15T08:19:22+00:00"
	rowtsTime, _ := time.Parse(time.RFC3339, rowts)

	rows := sqlmock.NewRows([]string{"aws_resources_id", "aws_ips_ip", "aws_hostnames_hostname", "is_public", "is_join", "ts", "aws_resources.account_id", "aws_resources.region", "aws_resources.type", "aws_resources.meta"}).AddRow("rid", "44.33.22.11", "yahoo.com", true, true, rowtsTime, "aid", "region", "type", []byte("{\"hi\":\"there1\"}"))
	mock.ExpectQuery("WITH").WithArgs(ipAddress, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByIP(context.Background(), at, ipAddress)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 1, len(results))
	assert.Equal(t, domain.CloudAssetDetails{
		PublicIPAddresses: []string{"44.33.22.11"},
		Hostnames:         []string{"yahoo.com"},
		ResourceType:      "type",
		AccountID:         "aid",
		Region:            "region",
		ARN:               "rid",
		Tags:              map[string]string{"hi": "there1"},
	}, results[0])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetIPsAtTimeMultiRows(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptNotFound,
	}

	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	ipAddress := "9.8.7.6"
	rowts1 := "2019-01-15T08:19:23+00:00"
	rowts1Time, _ := time.Parse(time.RFC3339, rowts1)
	rowts2 := "2019-01-15T08:55:41+00:00"
	rowts2Time, _ := time.Parse(time.RFC3339, rowts2)

	rows := sqlmock.NewRows([]string{"aws_resources_id", "aws_ips_ip", "aws_hostnames_hostname", "is_public", "is_join", "ts", "aws_resources.account_id", "aws_resources.region", "aws_resources.type", "aws_resources.meta"}).AddRow("rid", "44.33.22.11", "yahoo.com", true, true, rowts1Time, "aid", "region", "type", []byte("{\"hi\":\"there2\"}")).AddRow("rid2", "99.88.77.66", "google.com", true, true, rowts2Time, "aid2", "region2", "type2", []byte("{\"bye\":\"now\"}"))
	mock.ExpectQuery("WITH").WithArgs(ipAddress, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByIP(context.Background(), at, ipAddress)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{
			PublicIPAddresses: []string{"44.33.22.11"},
			Hostnames:         []string{"yahoo.com"},
			ResourceType:      "type",
			AccountID:         "aid",
			Region:            "region",
			ARN:               "rid",
			Tags:              map[string]string{"hi": "there2"},
		},
		domain.CloudAssetDetails{
			PublicIPAddresses: []string{"99.88.77.66"},
			Hostnames:         []string{"google.com"},
			ResourceType:      "type2",
			AccountID:         "aid2",
			Region:            "region2",
			ARN:               "rid2",
			Tags:              map[string]string{"bye": "now"},
		},
	}

	assertArrayEqualIgnoreOrder(t, expected, results)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetHostnamesAtTime(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptNotFound,
	}

	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	hostname := "google.com"
	rowts1 := "2019-01-15T08:19:24+00:00"
	rowts1Time, _ := time.Parse(time.RFC3339, rowts1)

	rows := sqlmock.NewRows([]string{"aws_resources_id", "aws_ips_ip", "aws_hostnames_hostname", "is_public", "is_join", "ts", "aws_resources.account_id", "aws_resources.region", "aws_resources.type", "aws_resources.meta"}).AddRow("rid", "44.33.22.11", "yahoo.com", true, true, rowts1Time, "aid", "region", "type", []byte("{\"hi\":\"there3\"}"))
	mock.ExpectQuery("WITH").WithArgs(hostname, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByHostname(context.Background(), at, hostname)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 1, len(results))
	assert.Equal(t, domain.CloudAssetDetails{
		PublicIPAddresses: []string{"44.33.22.11"},
		Hostnames:         []string{"yahoo.com"},
		ResourceType:      "type",
		AccountID:         "aid",
		Region:            "region",
		ARN:               "rid",
		Tags:              map[string]string{"hi": "there3"},
	}, results[0])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetHostnamesAtTimeMultiRows(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptNotFound,
	}

	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	hostname := "google.com"
	rowts1 := "2019-01-15T08:19:24+00:00"
	rowts1Time, _ := time.Parse(time.RFC3339, rowts1)

	rows := sqlmock.NewRows([]string{"aws_resources_id", "aws_ips_ip", "aws_hostnames_hostname", "is_public", "is_join", "ts", "aws_resources.account_id", "aws_resources.region", "aws_resources.type", "aws_resources.meta"}).AddRow("rid", "44.33.22.11", "yahoo.com", true, true, rowts1Time, "aid", "region", "type", []byte("{\"hi\":\"there3\"}")).AddRow("rid", "9.8.7.6", "yahoo.com", true, true, rowts1Time, "aid", "region", "type", []byte("{\"hi\":\"there4\"}")).AddRow("rid", "9.8.7.6", "yahoo.com", true, true, rowts1Time, "aid", "region", "type", []byte("{\"hi\":\"there5\"}")).AddRow("rid", "9.8.7.6", nil, false, true, rowts1Time, "aid", "region", "type", []byte("{\"hi\":\"there5\"}"))
	mock.ExpectQuery("WITH").WithArgs(hostname, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByHostname(context.Background(), at, hostname)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 1, len(results))
	assert.Equal(t, domain.CloudAssetDetails{
		PrivateIPAddresses: []string{"9.8.7.6"},
		PublicIPAddresses:  []string{"44.33.22.11", "9.8.7.6"},
		Hostnames:          []string{"yahoo.com"},
		ResourceType:       "type",
		AccountID:          "aid",
		Region:             "region",
		ARN:                "rid",
		Tags:               map[string]string{"hi": "there3"},
	}, results[0])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestScriptNotFound(t *testing.T) {
	mockdb, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockdb.Close()

	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptNotFound,
	}

	require.Error(t, thedb.RunScript(context.Background(), "script1"))
}

func TestRunScriptTxFailBegin(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockdb.Close()

	mock.ExpectBegin().WillReturnError(errors.New("tx fail"))
	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
	}

	require.Error(t, thedb.RunScript(context.Background(), "script1"))
}

func TestRunScriptTxRollbackOnFail(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockdb.Close()

	mock.ExpectBegin()
	mock.ExpectExec(scriptText).WillReturnError(errors.New("bad query"))
	mock.ExpectRollback()
	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
	}

	require.Error(t, thedb.RunScript(context.Background(), "script1"))
}

func TestRunScriptTxRollbackFail(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockdb.Close()

	mock.ExpectBegin()
	mock.ExpectExec(scriptText).WillReturnError(errors.New("bad query"))
	mock.ExpectRollback().WillReturnError(errors.New("bad rollback"))
	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
	}

	require.Error(t, thedb.RunScript(context.Background(), "script1"))
}

func TestRunScriptTxCommit(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockdb.Close()

	mock.ExpectBegin()
	mock.ExpectExec(scriptText).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	thedb := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
	}
	require.NoError(t, thedb.RunScript(context.Background(), "script1"))
}

func TestGeneratePartitionNeedOne(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	// within 3 days of latestPartitionEnd; so we know we need a new partition soon
	createdAt := time.Date(2019, 9, 29, 0, 0, 0, 0, time.UTC)

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return createdAt },
	}

	latestPartitionBegin, _ := time.Parse(time.RFC3339, "2019-04-01T00:00:00Z")
	latestPartitionEnd, _ := time.Parse(time.RFC3339, "2019-10-01T00:00:00Z")
	newEnd, _ := time.Parse(time.RFC3339, "2019-12-30T00:00:00Z")
	nextPartition := "aws_events_ips_hostnames_2019_10_01to2019_12_30"
	rows := sqlmock.NewRows([]string{"partition_begin", "partition_end"}).AddRow(latestPartitionBegin, latestPartitionEnd)
	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT").WillReturnRows(rows).RowsWillBeClosed()
	mock.ExpectExec("INSERT").WithArgs(nextPartition, createdAt, latestPartitionEnd, newEnd).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + nextPartition + " PARTITION OF aws_events_ips_hostnames FOR VALUES FROM").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = db.GeneratePartition(context.Background(), time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGeneratePartitionNeedOneButShouldNotCreateItYet(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	// further than 3 days of latestPartitionEnd; so we know we need a new partition but we don't try to create it yet
	createdAt := time.Date(2019, 9, 10, 0, 0, 0, 0, time.UTC)

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return createdAt },
	}

	latestPartitionBegin, _ := time.Parse(time.RFC3339, "2019-04-01T00:00:00Z")
	latestPartitionEnd, _ := time.Parse(time.RFC3339, "2019-10-01T00:00:00Z")
	rows := sqlmock.NewRows([]string{"partition_begin", "partition_end"}).AddRow(latestPartitionBegin, latestPartitionEnd)
	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT").WillReturnRows(rows).RowsWillBeClosed()
	mock.ExpectCommit()

	err = db.GeneratePartition(context.Background(), time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGeneratePartitionAlreadyExists(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	createdAt := time.Date(2019, 03, 28, 0, 0, 0, 0, time.UTC) // date is _before_ the already-existing future partition

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return createdAt },
	}

	latestPartitionBegin, _ := time.Parse(time.RFC3339, "2019-04-01T00:00:00Z")
	latestPartitionEnd, _ := time.Parse(time.RFC3339, "2019-10-01T00:00:00Z")
	rows := sqlmock.NewRows([]string{"partition_begin", "partition_end"}).AddRow(latestPartitionBegin, latestPartitionEnd)
	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT").WillReturnRows(rows).RowsWillBeClosed()
	mock.ExpectCommit()

	err = db.GeneratePartition(context.Background(), time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGeneratePartitionFirstTime(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	createdAt := time.Date(2019, time.May, 5, 0, 0, 0, 0, time.UTC)

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now: func() time.Time {
			return createdAt
		},
	}

	begin, _ := time.Parse(time.RFC3339, "2019-05-05T00:00:00Z")
	end, _ := time.Parse(time.RFC3339, "2019-08-03T00:00:00Z")

	nextPartition := "aws_events_ips_hostnames_2019_05_05to2019_08_03"
	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows).RowsWillBeClosed()
	mock.ExpectExec("INSERT").WithArgs(nextPartition, createdAt, begin, end).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + nextPartition + " PARTITION OF aws_events_ips_hostnames FOR VALUES FROM").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = db.GeneratePartition(context.Background(), time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGeneratePartitionScanError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now: func() time.Time {
			return time.Date(2019, time.May, 1, 0, 0, 0, 0, time.UTC)
		},
	}

	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT").WillReturnError(errors.New("")).RowsWillBeClosed()

	err = db.GeneratePartition(context.Background(), time.Time{}, 0)
	assert.Error(t, err)
}

func TestGeneratePartitionInvalidPartition(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
	}

	rows := sqlmock.NewRows([]string{"partition_begin", "partition_end"}).AddRow("not a valid date", "also invalid")
	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT").WillReturnRows(rows).RowsWillBeClosed()

	err = db.GeneratePartition(context.Background(), time.Time{}, 0)
	assert.Error(t, err)
}

func TestGeneratePartitionTxError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	createdAt := time.Date(2019, 03, 03, 12, 12, 12, 0, time.UTC)

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return createdAt },
	}

	mock.ExpectBegin().WillReturnError(errors.New(""))

	err = db.GeneratePartition(context.Background(), time.Time{}, 0)
	assert.Error(t, err)
}

func TestGeneratePartitionLockError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	createdAt := time.Date(2019, 03, 03, 12, 12, 12, 0, time.UTC)

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return createdAt },
	}

	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnError(errors.New(""))

	err = db.GeneratePartition(context.Background(), time.Time{}, 0)
	assert.Error(t, err)
}

func TestGeneratePartitionInsertFailure(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	createdAt := time.Date(2019, 03, 03, 12, 12, 12, 0, time.UTC)

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return createdAt },
	}

	latestPartitionBegin, _ := time.Parse(time.RFC3339, "2019-03-01T00:00:00Z")
	latestPartitionEnd, _ := time.Parse(time.RFC3339, "2019-04-01T00:00:00Z")
	newEnd, _ := time.Parse(time.RFC3339, "2019-07-01T00:00:00Z")
	nextPartition := "aws_events_ips_hostnames_2019_04to2019_07" // next quarter
	rows := sqlmock.NewRows([]string{"partition_begin", "partition_end"}).AddRow(latestPartitionBegin, latestPartitionEnd)
	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT").WillReturnRows(rows).RowsWillBeClosed()
	mock.ExpectExec("INSERT").WithArgs(nextPartition, time.RFC3339, latestPartitionEnd, newEnd).WillReturnError(errors.New(""))
	mock.ExpectRollback()

	err = db.GeneratePartition(context.Background(), time.Time{}, 0)
	assert.Error(t, err)
}

func TestGeneratePartitionConflict(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	createdAt := time.Date(2019, 03, 03, 0, 0, 0, 0, time.UTC)

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return createdAt },
	}

	latestPartitionBegin, _ := time.Parse(time.RFC3339, "2019-03-01T00:00:00Z")
	latestPartitionEnd, _ := time.Parse(time.RFC3339, "2019-03-01T00:00:00Z")
	newEnd, _ := time.Parse(time.RFC3339, "2019-05-30T00:00:00Z")
	nextPartition := "aws_events_ips_hostnames_2019_03_01to2019_05_30" // next quarter
	rows := sqlmock.NewRows([]string{"partition_begin", "partition_end"}).AddRow(latestPartitionBegin, latestPartitionEnd)
	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT").WillReturnRows(rows).RowsWillBeClosed()
	mock.ExpectExec("INSERT").WithArgs(nextPartition, createdAt, latestPartitionEnd, newEnd).WillReturnResult(sqlmock.NewResult(0, 0))

	err = db.GeneratePartition(context.Background(), time.Time{}, 0)
	assert.Error(t, err)
	_, ok := err.(domain.PartitionConflict)
	assert.True(t, ok, fmt.Sprintf("Expected a PartitionConflict, but received %t", err))
}

func TestGeneratePartitionCreateFailure(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	createdAt := time.Date(2019, 03, 03, 12, 12, 12, 0, time.UTC)

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return createdAt },
	}

	latestPartitionBegin, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	latestPartitionEnd, _ := time.Parse(time.RFC3339, "2019-02-01T00:00:00Z")
	newEnd, _ := time.Parse(time.RFC3339, "2019-05-01T00:00:00Z")
	nextPartition := "aws_events_ips_hostnames_2019_02to2019_05"
	rows := sqlmock.NewRows([]string{"partition_begin", "partition_end"}).AddRow(latestPartitionBegin, latestPartitionEnd)
	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT").WillReturnRows(rows).RowsWillBeClosed()
	mock.ExpectExec("INSERT").WithArgs(nextPartition, createdAt, latestPartitionEnd, newEnd).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + nextPartition + " PARTITION OF aws_events_ips_hostnames FOR VALUES FROM").
		WillReturnError(errors.New(""))
	mock.ExpectRollback()

	err = db.GeneratePartition(context.Background(), time.Time{}, 0)
	assert.Error(t, err)
}

func TestGeneratePartitionWithTimestamp(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	createdAt := time.Date(2019, 03, 03, 0, 0, 0, 0, time.UTC)

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return createdAt },
	}

	ts, _ := time.Parse(time.RFC3339, "2019-10-01T00:00:00Z")
	newEnd, _ := time.Parse(time.RFC3339, "2019-12-30T00:00:00Z")
	nextPartition := "aws_events_ips_hostnames_2019_10_01to2019_12_30"
	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT").WithArgs(nextPartition, createdAt, ts, newEnd).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + nextPartition + " PARTITION OF aws_events_ips_hostnames FOR VALUES FROM").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = db.GeneratePartition(context.Background(), ts, 0)
	assert.NoError(t, err)
}

func TestGeneratePartitionWithTimestampAndDuration(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	createdAt := time.Date(2019, 03, 03, 0, 0, 0, 0, time.UTC)
	days := 14

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return createdAt },
	}

	ts, _ := time.Parse(time.RFC3339, "2019-10-01T00:00:00Z")
	newEnd, _ := time.Parse(time.RFC3339, "2019-10-15T00:00:00Z")
	nextPartition := "aws_events_ips_hostnames_2019_10_01to2019_10_15"
	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT").WithArgs(nextPartition, createdAt, ts, newEnd).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + nextPartition + " PARTITION OF aws_events_ips_hostnames FOR VALUES FROM").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = db.GeneratePartition(context.Background(), ts, days)
	assert.NoError(t, err)
}

func TestGetPartitionsScanError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
	}

	createdAt, _ := time.Parse(time.RFC3339, "2019-03-31T00:00:00Z")
	partitionBegin, _ := time.Parse(time.RFC3339, "2019-04-01T00:00:00Z")
	partitionEnd, _ := time.Parse(time.RFC3339, "2019-07-01T00:00:00Z")
	rows := sqlmock.NewRows([]string{"created_at", "partition_begin", "partition_end"}).AddRow(createdAt, partitionBegin, partitionEnd) // missing name
	mock.ExpectQuery("SELECT").WillReturnRows(rows).RowsWillBeClosed()

	_, err = db.GetPartitions(context.Background())
	assert.Error(t, err)
}

func TestGetPartitions(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
	}
	name := "partition"
	createdAt, _ := time.Parse(time.RFC3339, "2019-03-31T00:00:00Z")
	partitionBegin, _ := time.Parse(time.RFC3339, "2019-04-01T00:00:00Z")
	partitionEnd, _ := time.Parse(time.RFC3339, "2019-07-01T00:00:00Z")
	rows := sqlmock.NewRows([]string{"name", "created_at", "partition_begin", "partition_end"}).AddRow(name, createdAt, partitionBegin, partitionEnd)
	mock.ExpectQuery("SELECT").WillReturnRows(rows).RowsWillBeClosed()
	row := sqlmock.NewRows([]string{"c"}).AddRow(10)
	mock.ExpectQuery("SELECT").WillReturnRows(row).RowsWillBeClosed()
	results, err := db.GetPartitions(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, 10, results[0].Count)
}

const (
	testPartitionName = "TEST_PARTITION"
)

func TestDeletePartitions(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	now := time.Date(2019, 9, 29, 0, 0, 0, 0, time.UTC)
	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return now },
	}

	partitions := []string{"one", "two", testPartitionName}
	rows := sqlmock.NewRows([]string{"name"})
	for _, p := range partitions {
		rows = rows.AddRow(p)
	}

	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE").WithArgs(testPartitionName).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(fmt.Sprintf("DROP TABLE %s", testPartitionName)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = db.DeletePartitions(context.Background(), testPartitionName)
	assert.NoError(t, err)
}

func TestDeletePartitionsTxError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	now := time.Date(2019, 9, 29, 0, 0, 0, 0, time.UTC)
	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return now },
	}

	partitions := []string{"one", "two", testPartitionName}
	rows := sqlmock.NewRows([]string{"name"})
	for _, p := range partitions {
		rows = rows.AddRow(p)
	}
	mock.ExpectBegin().WillReturnError(errors.New(""))

	err = db.DeletePartitions(context.Background(), testPartitionName)
	assert.Error(t, err)
}

func TestDeletePartitionsLockError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	now := time.Date(2019, 9, 29, 0, 0, 0, 0, time.UTC)
	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return now },
	}

	partitions := []string{"one", "two", testPartitionName}
	rows := sqlmock.NewRows([]string{"name"})
	for _, p := range partitions {
		rows = rows.AddRow(p)
	}
	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnError(errors.New(""))
	mock.ExpectRollback()

	err = db.DeletePartitions(context.Background(), testPartitionName)
	assert.Error(t, err)
}

func TestDeletePartitionsDeleteError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	now := time.Date(2019, 9, 29, 0, 0, 0, 0, time.UTC)
	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return now },
	}

	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE").WithArgs(testPartitionName).WillReturnError(errors.New(""))
	mock.ExpectRollback()

	err = db.DeletePartitions(context.Background(), testPartitionName)
	assert.Error(t, err)
}

func TestDeletePartitionsDropError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	now := time.Date(2019, 9, 29, 0, 0, 0, 0, time.UTC)
	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return now },
	}

	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE").WithArgs(testPartitionName).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DROP TABLE " + testPartitionName).WillReturnError(errors.New(""))
	mock.ExpectRollback()

	err = db.DeletePartitions(context.Background(), testPartitionName)
	assert.Error(t, err)
}

func TestDeletePartitionsNotFoundError(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	now := time.Date(2019, 9, 29, 0, 0, 0, 0, time.UTC)
	db := DB{
		sqldb:   mockdb,
		scripts: scriptFound,
		now:     func() time.Time { return now },
	}

	nonexistentPartition := "UNREAL_PARTITION"

	mock.ExpectBegin()
	mock.ExpectExec("LOCK").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE").WithArgs(nonexistentPartition).WillReturnResult(sqlmock.NewResult(1, 0))
	mock.ExpectRollback()

	err = db.DeletePartitions(context.Background(), nonexistentPartition)
	require.Error(t, err)
	_, ok := err.(domain.NotFoundPartition)
	assert.True(t, ok)
}

func fakeCloudAssetChanges() domain.CloudAssetChanges {
	privateIPs := []string{"4.3.2.1"}
	publicIPs := []string{"8.7.6.5"}
	hostnames := []string{"google.com"}
	networkChangesArray := []domain.NetworkChanges{
		domain.NetworkChanges{
			PrivateIPAddresses: privateIPs,
			PublicIPAddresses:  publicIPs,
			Hostnames:          hostnames,
			ChangeType:         "ADDED",
		},
	}
	timestamp, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	cloudAssetChanges := domain.CloudAssetChanges{
		Changes:      networkChangesArray,
		ChangeTime:   timestamp,
		ResourceType: "rtype",
		AccountID:    "aid",
		Region:       "region",
		ARN:          "arn",
		Tags:         map[string]string{"tag1": "val1"},
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

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

	postgresConfig := PostgresConfig{"this is not a hostname", "99", "me!", "mypassword!", "name"}

	if err := thedb.Init(context.Background(), &postgresConfig); err == nil {
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
	mock.ExpectExec("INSERT INTO " + tableAWSHostnames).WithArgs("google.com").WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectExec("INSERT INTO " + tableAWSIPS).WithArgs("4.3.2.1").WillReturnResult(sqlmock.NewResult(1, 1))          // nolint
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + fmt.Sprintf("%s_2019_04to06", tableAWSEventsIPSHostnames) + " PARTITION OF " + tableAWSEventsIPSHostnames + " FOR VALUES FROM \\('2019-04-01'\\) TO \\('2019-06-30'\\);").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO "+tableAWSEventsIPSHostnames).WithArgs(timestamp, false, true, "arn", "4.3.2.1", nil).WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectExec("INSERT INTO " + tableAWSIPS).WithArgs("8.7.6.5").WillReturnResult(sqlmock.NewResult(1, 1))                                                  // nolint
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + fmt.Sprintf("%s_2019_04to06", tableAWSEventsIPSHostnames) + " PARTITION OF " + tableAWSEventsIPSHostnames + " FOR VALUES FROM \\('2019-04-01'\\) TO \\('2019-06-30'\\);").WillReturnResult(sqlmock.NewResult(1, 1))
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
	assert.Equal(t, domain.CloudAssetDetails{nil, []string{"44.33.22.11"}, []string{"yahoo.com"}, "type", "aid", "region", "rid", map[string]string{"hi": "there1"}}, results[0])

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
		domain.CloudAssetDetails{nil, []string{"44.33.22.11"}, []string{"yahoo.com"}, "type", "aid", "region", "rid", map[string]string{"hi": "there2"}},    // nolint
		domain.CloudAssetDetails{nil, []string{"99.88.77.66"}, []string{"google.com"}, "type2", "aid2", "region2", "rid2", map[string]string{"bye": "now"}}, // nolint
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
	assert.Equal(t, domain.CloudAssetDetails{nil, []string{"44.33.22.11"}, []string{"yahoo.com"}, "type", "aid", "region", "rid", map[string]string{"hi": "there3"}}, results[0])

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
	assert.Equal(t, domain.CloudAssetDetails{[]string{"9.8.7.6"}, []string{"44.33.22.11", "9.8.7.6"}, []string{"yahoo.com"}, "type", "aid", "region", "rid", map[string]string{"hi": "there3"}}, results[0])

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

func fakeCloudAssetChanges() domain.CloudAssetChanges {
	privateIPs := []string{"4.3.2.1"}
	publicIPs := []string{"8.7.6.5"}
	hostnames := []string{"google.com"}
	networkChangesArray := []domain.NetworkChanges{domain.NetworkChanges{privateIPs, publicIPs, hostnames, "ADDED"}}
	timestamp, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	cloudAssetChanges := domain.CloudAssetChanges{networkChangesArray, timestamp, "rtype", "aid", "region", "arn", map[string]string{"tag1": "val1"}}
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

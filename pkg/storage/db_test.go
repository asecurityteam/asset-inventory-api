package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/logevent"
	"github.com/asecurityteam/runhttp"
)

func TestGracefulHandlingOfTxBeginFailure(t *testing.T) {
	// no panics, in other words
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{mockdb, stdoutLogger, sync.Once{}}

	mock.ExpectBegin().WillReturnError(fmt.Errorf("could not start transaction"))

	if err = thedb.StoreCloudAsset(context.Background(), fakeCloudAssetChanges()); err == nil {
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

	thedb := DB{mockdb, stdoutLogger, sync.Once{}}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("arn", "aid", "region", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnResult(sqlmock.NewResult(1, 1))

	fakeContext, _ := mockdb.BeginTx(context.Background(), nil);

	if err = thedb.saveResource(context.Background(), fakeCloudAssetChanges(), fakeContext); err != nil {
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

	thedb := DB{mockdb, stdoutLogger, sync.Once{}}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("arn", "aid", "region", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnError(fmt.Errorf("some error"))
	mock.ExpectRollback()

	if err = thedb.StoreCloudAsset(context.Background(), fakeCloudAssetChanges()); err == nil {
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

	thedb := DB{mockdb, stdoutLogger, sync.Once{}}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("arn", "aid", "region", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnResult(sqlmock.NewResult(1, 1))
	timestamp, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	mock.ExpectExec("INSERT INTO " + tableAWSHostnames).WithArgs("google.com").WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectExec("INSERT INTO " + tableAWSIPS).WithArgs("4.3.2.1").WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + fmt.Sprintf("%s_2019_04to06", tableAWSEventsIPSHostnames) + " PARTITION OF " + tableAWSEventsIPSHostnames + " FOR VALUES FROM \\('2019-04-01'\\) TO \\('2019-06-30'\\);").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("CREATE INDEX IF NOT EXISTS " + fmt.Sprintf("%s_2019_04to06_aws_ips_ip_ts_idx", tableAWSEventsIPSHostnames)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO "+tableAWSEventsIPSHostnames).WithArgs(timestamp, false, true, "arn", "4.3.2.1", nil).WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectExec("INSERT INTO " + tableAWSIPS).WithArgs("8.7.6.5").WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + fmt.Sprintf("%s_2019_04to06", tableAWSEventsIPSHostnames) + " PARTITION OF " + tableAWSEventsIPSHostnames + " FOR VALUES FROM \\('2019-04-01'\\) TO \\('2019-06-30'\\);").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("CREATE INDEX IF NOT EXISTS " + fmt.Sprintf("%s_2019_04to06_aws_ips_ip_ts_idx", tableAWSEventsIPSHostnames)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO "+tableAWSEventsIPSHostnames).WithArgs(timestamp, true, true, "arn", "8.7.6.5", "google.com").WillReturnResult(sqlmock.NewResult(1, 1)) // nolint

	if err = thedb.StoreCloudAsset(context.Background(), fakeCloudAssetChanges()); err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetIPsForTimeRange(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{mockdb, stdoutLogger, sync.Once{}}

	from, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	to, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	rowts := "2019-01-15T08:19:22+00:00"
	rowtsTime, _ := time.Parse(time.RFC3339, rowts)

	rows := sqlmock.NewRows([]string{"aws_resources_id", "aws_ips_ip", "aws_hostnames_hostname", "is_public", "is_join", "ts", "aws_resources.account_id", "aws_resources.region", "aws_resources.type", "aws_resources.meta"}).AddRow("rid", "44.33.22.11", "yahoo.com", true, true, rowtsTime, "aid", "region", "type", []byte("{\"hi\":\"there1\"}"))
	mock.ExpectQuery("SELECT").WithArgs(from, to).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.GetIPAddressesForTimeRange(context.Background(), from, to)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 1, len(results))
	assert.Equal(t, domain.NetworkChangeEvent{"rid", "44.33.22.11", sql.NullString{"yahoo.com", true}, true, true, rowtsTime, "aid", "region", "type", map[string]string{"hi": "there1"}}, results[0])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetIPsByIP(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{mockdb, stdoutLogger, sync.Once{}}

	ipAddress := "9.8.7.6"
	rowts1 := "2019-01-15T08:19:23+00:00"
	rowts1Time, _ := time.Parse(time.RFC3339, rowts1)
	rowts2 := "2019-01-15T08:55:41+00:00"
	rowts2Time, _ := time.Parse(time.RFC3339, rowts2)

	rows := sqlmock.NewRows([]string{"aws_resources_id", "aws_ips_ip", "aws_hostnames_hostname", "is_public", "is_join", "ts", "aws_resources.account_id", "aws_resources.region", "aws_resources.type", "aws_resources.meta"}).AddRow("rid", "44.33.22.11", "yahoo.com", true, true, rowts1Time, "aid", "region", "type", []byte("{\"hi\":\"there2\"}")).AddRow("rid2", "99.88.77.66", "google.com", true, true, rowts2Time, "aid2", "region2", "type2", []byte("{\"bye\":\"now\"}"))
	mock.ExpectQuery("SELECT").WithArgs(ipAddress).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.GetIPAddressesForIPAddress(context.Background(), ipAddress)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 2, len(results))
	assert.Equal(t, domain.NetworkChangeEvent{"rid", "44.33.22.11", sql.NullString{"yahoo.com", true}, true, true, rowts1Time, "aid", "region", "type", map[string]string{"hi": "there2"}}, results[0])
	assert.Equal(t, domain.NetworkChangeEvent{"rid2", "99.88.77.66", sql.NullString{"google.com", true}, true, true, rowts2Time, "aid2", "region2", "type2", map[string]string{"bye": "now"}}, results[1])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetIPsByHostname(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{mockdb, stdoutLogger, sync.Once{}}

	hostname := "google.com"
	rowts1 := "2019-01-15T08:19:24+00:00"
	rowts1Time, _ := time.Parse(time.RFC3339, rowts1)

	rows := sqlmock.NewRows([]string{"aws_resources_id", "aws_ips_ip", "aws_hostnames_hostname", "is_public", "is_join", "ts", "aws_resources.account_id", "aws_resources.region", "aws_resources.type", "aws_resources.meta"}).AddRow("rid", "44.33.22.11", "yahoo.com", true, true, rowts1Time, "aid", "region", "type", []byte("{\"hi\":\"there3\"}"))
	mock.ExpectQuery("SELECT").WithArgs(hostname).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.GetIPAddressesForHostname(context.Background(), hostname)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 1, len(results))
	assert.Equal(t, domain.NetworkChangeEvent{"rid", "44.33.22.11", sql.NullString{"yahoo.com", true}, true, true, rowts1Time, "aid", "region", "type", map[string]string{"hi": "there3"}}, results[0])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func stdoutLogger(_ context.Context) runhttp.Logger {
	return logevent.New(logevent.Config{Output: os.Stdout})
}

func fakeCloudAssetChanges() domain.CloudAssetChanges {
	privateIPs := []string{"4.3.2.1"}
	publicIPs := []string{"8.7.6.5"}
	hostnames := []string{"google.com"}
	networkChangesArray := []domain.NetworkChanges{domain.NetworkChanges{privateIPs, publicIPs, hostnames, "ADDED"}}
	timestamp, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	cloudAssetChanges := domain.CloudAssetChanges{networkChangesArray, timestamp, "rtype", "aid", "region", "rid", "arn", map[string]string{"tag1": "val1"}}
	return cloudAssetChanges
}

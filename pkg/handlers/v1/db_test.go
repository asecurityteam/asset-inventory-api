package v1

import (
	"context"
	"fmt"
	"os"
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

	thedb := DB{mockdb, stdoutLogger}

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

	thedb := DB{mockdb, stdoutLogger}

	mock.ExpectExec("INSERT INTO").WithArgs("aws_resources", "arn", "aid", "region", "{\"tag1\":\"val1\"}").WillReturnResult(sqlmock.NewResult(1, 1))

	if err = thedb.saveResource(context.Background(), fakeCloudAssetChanges()); err != nil {
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

	thedb := DB{mockdb, stdoutLogger}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("aws_resources", "arn", "aid", "region", "{\"tag1\":\"val1\"}").WillReturnError(fmt.Errorf("some error"))
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

	thedb := DB{mockdb, stdoutLogger}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs(tableAWSResources, "arn", "aid", "region", "{\"tag1\":\"val1\"}").WillReturnResult(sqlmock.NewResult(1, 1))
	timestamp, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	mock.ExpectExec("INSERT INTO").WithArgs(tableAWSHostnames, timestamp, "google.com", "arn").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO").WithArgs(tableAWSIPS, timestamp, "4.3.2.1", false, true, "arn").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO").WithArgs(tableAWSIPS, timestamp, "8.7.6.5", true, true, "arn").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO").WithArgs(tableAWSIPSHostnames, timestamp, "google.com", "8.7.6.5").WillReturnResult(sqlmock.NewResult(1, 1))

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

	thedb := DB{mockdb, stdoutLogger}

	from, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	to, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	rowts := "2019-01-15T08:19:23+00:00"
	rowtsTime, _ := time.Parse(time.RFC3339, rowts)

	rows := sqlmock.NewRows([]string{"hostname", "ip", "is_public", "is_join", "ts"}).AddRow("yahoo.com", "44.33.22.11", true, true, rowts)
	mock.ExpectQuery("SELECT").WithArgs(from, to).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.GetIPAddressesForTimeRange(context.Background(), from, to)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 1, len(results))
	assert.Equal(t, domain.QueryResult{"yahoo.com", "44.33.22.11", true, true, rowtsTime}, results[0])

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

	thedb := DB{mockdb, stdoutLogger}

	ipAddress := "9.8.7.6"
	rowts1 := "2019-01-15T08:19:23+00:00"
	rowts1Time, _ := time.Parse(time.RFC3339, rowts1)
	rowts2 := "2019-01-15T08:55:41+00:00"
	rowts2Time, _ := time.Parse(time.RFC3339, rowts2)

	rows := sqlmock.NewRows([]string{"hostname", "ip", "is_public", "is_join", "ts"}).AddRow("yahoo.com", "44.33.22.11", true, true, rowts1).AddRow("google.com", "99.88.77.66", true, true, rowts2)
	mock.ExpectQuery("SELECT").WithArgs(ipAddress).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.GetIPAddressesForIPAddress(context.Background(), ipAddress)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 2, len(results))
	assert.Equal(t, domain.QueryResult{"yahoo.com", "44.33.22.11", true, true, rowts1Time}, results[0])
	assert.Equal(t, domain.QueryResult{"google.com", "99.88.77.66", true, true, rowts2Time}, results[1])

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

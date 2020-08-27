package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
)

func TestDBInitHandleOpenError(t *testing.T) {
	thedb := DB{}

	url := "this is not valid database url"
	partitionTTL := 2

	if err := thedb.Init(context.Background(), url, partitionTTL); err == nil {
		t.Errorf("DB.Init should have returned a non-nil error")
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
		sqldb: mockdb,
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
		sqldb: mockdb,
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
		sqldb: mockdb,
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
		sqldb: mockdb,
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

	theDB := DB{
		sqldb: mockdb,
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

	if err = theDB.Store(ctx, fakeCloudAssetChanges()); err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func withSchemaVersion(version uint, mock sqlmock.Sqlmock) {
	rows := sqlmock.NewRows([]string{"version"}).AddRow(version)
	mock.ExpectQuery("select version from schema_migrations").WillReturnRows(rows).RowsWillBeClosed()
}

func TestGetIPsAtTimeLegacySchema(t *testing.T) {
	//version := uint(1)
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := &DB{
		sqldb: mockdb,
	}

	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	ipAddress := "9.8.7.6" //nolint
	rowts := "2019-01-15T08:19:22+00:00"
	rowtsTime, _ := time.Parse(time.RFC3339, rowts)

	withSchemaVersion(1, mock)
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

func TestGetIPsAtTimeMultiRowsLegacySchema(t *testing.T) {

	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := &DB{
		sqldb: mockdb,
	}

	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	ipAddress := "9.8.7.6"
	rowts1 := "2019-01-15T08:19:23+00:00"
	rowts1Time, _ := time.Parse(time.RFC3339, rowts1)
	rowts2 := "2019-01-15T08:55:41+00:00"
	rowts2Time, _ := time.Parse(time.RFC3339, rowts2)

	withSchemaVersion(1, mock)

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

func TestGetPrivateIPsAtTimeM1Schema(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := &DB{
		sqldb: mockdb,
	}

	withSchemaVersion(4, mock)

	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	ipAddress := "10.2.2.6"
	rows := sqlmock.NewRows([]string{"aws_private_ip_assignment_private_ip",
		"aws_resource_arn_id",
		"aws_resource_meta",
		"aws_region_region",
		"aws_resource_type_resource_type",
		"aws_account_account",
		"aws_account_id",
		"aws_account_account",
		"owner_login",
		"owner_email",
		"owner_name",
		"owner_valid",
		"champion_login",
		"champion_email",
		"champion_name",
		"champion_valid"}).AddRow("10.2.2.6",
		"rid",
		[]byte("{\"hi\":\"there1\"}"),
		"region",
		"type",
		"aid",
		"1",
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true)

	mock.ExpectQuery("select").WithArgs(ipAddress, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByIP(context.Background(), at, ipAddress)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 1, len(results))
	assert.Equal(t, domain.CloudAssetDetails{
		PrivateIPAddresses: []string{"10.2.2.6"},
		ResourceType:       "type",
		AccountID:          "aid",
		Region:             "region",
		ARN:                "rid",
		Tags:               map[string]string{"hi": "there1"},
		AccountOwner: domain.AccountOwner{
			AccountID: "aid",
			Owner: domain.Person{
				Login: "login",
				Email: "email@atlassian.com",
				Name:  "name",
				Valid: true,
			},
			Champions: []domain.Person{
				{
					Login: "login2",
					Email: "email2@atlassian.com",
					Name:  "name2",
					Valid: true,
				},
			},
		},
	}, results[0])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetPublicIPsAtTimeM1Schema(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := &DB{
		sqldb: mockdb,
	}

	withSchemaVersion(4, mock)

	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	ipAddress := "9.8.7.6"
	rows := sqlmock.NewRows([]string{"aws_public_ip_assignment_public_ip",
		"aws_public_ip_assignment_hostname",
		"aws_resource_arn_id",
		"aws_resource_meta",
		"aws_region_region",
		"aws_resource_type_resource_type",
		"aws_account_account",
		"aws_account_id",
		"aws_account_account",
		"owner_login",
		"owner_email",
		"owner_name",
		"owner_valid",
		"champion_login",
		"champion_email",
		"champion_name",
		"champion_valid"}).AddRow("9.8.7.6",
		"yahoo.com",
		"rid",
		[]byte("{\"hi\":\"there1\"}"),
		"region",
		"type",
		"aid",
		"1",
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true)

	mock.ExpectQuery("select").WithArgs(ipAddress, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByIP(context.Background(), at, ipAddress)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 1, len(results))
	assert.Equal(t, domain.CloudAssetDetails{
		PublicIPAddresses: []string{"9.8.7.6"},
		Hostnames:         []string{"yahoo.com"},
		ResourceType:      "type",
		AccountID:         "aid",
		Region:            "region",
		ARN:               "rid",
		Tags:              map[string]string{"hi": "there1"},
		AccountOwner: domain.AccountOwner{
			AccountID: "aid",
			Owner: domain.Person{
				Login: "login",
				Email: "email@atlassian.com",
				Name:  "name",
				Valid: true,
			},
			Champions: []domain.Person{
				{
					Login: "login2",
					Email: "email2@atlassian.com",
					Name:  "name2",
					Valid: true,
				},
			},
		},
	}, results[0])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetPrivateIPsAtTimeMultiRowsM1Schema(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := &DB{
		sqldb: mockdb,
	}

	withSchemaVersion(4, mock)
	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	ipAddress := "172.16.2.2"
	rows := sqlmock.NewRows([]string{"aws_private_ip_assignment_private_ip",
		"aws_resource_arn_id",
		"aws_resource_meta",
		"aws_region_region",
		"aws_resource_type_resource_type",
		"aws_account_account",
		"aws_account_id",
		"aws_account_account",
		"owner_login",
		"owner_email",
		"owner_name",
		"owner_valid",
		"champion_login",
		"champion_email",
		"champion_name",
		"champion_valid"}).AddRow("172.16.2.2",
		"rid",
		[]byte("{\"hi\":\"there1\"}"),
		"region",
		"type",
		"aid",
		"1",
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true).AddRow("172.16.3.3",
		"rid2",
		[]byte("{\"bye\":\"now\"}"),
		"region2",
		"type2",
		"aid2",
		"2",
		"aid2",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true)

	mock.ExpectQuery("select").WithArgs(ipAddress, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByIP(context.Background(), at, ipAddress)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{
			PrivateIPAddresses: []string{"172.16.2.2"},
			ResourceType:       "type",
			AccountID:          "aid",
			Region:             "region",
			ARN:                "rid",
			Tags:               map[string]string{"hi": "there1"},
			AccountOwner: domain.AccountOwner{
				AccountID: "aid",
				Owner: domain.Person{
					Login: "login",
					Email: "email@atlassian.com",
					Name:  "name",
					Valid: true,
				},
				Champions: []domain.Person{
					{
						Login: "login2",
						Email: "email2@atlassian.com",
						Name:  "name2",
						Valid: true,
					},
				},
			},
		},
		domain.CloudAssetDetails{
			PrivateIPAddresses: []string{"172.16.3.3"},
			ResourceType:       "type2",
			AccountID:          "aid2",
			Region:             "region2",
			ARN:                "rid2",
			Tags:               map[string]string{"bye": "now"},
			AccountOwner: domain.AccountOwner{
				AccountID: "aid2",
				Owner: domain.Person{
					Login: "login",
					Email: "email@atlassian.com",
					Name:  "name",
					Valid: true,
				},
				Champions: []domain.Person{
					{
						Login: "login2",
						Email: "email2@atlassian.com",
						Name:  "name2",
						Valid: true,
					},
				},
			},
		},
	}

	assertArrayEqualIgnoreOrder(t, expected, results)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetPublicIPsAtTimeMultiRowsM1Schema(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := &DB{
		sqldb: mockdb,
	}

	withSchemaVersion(4, mock)
	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	ipAddress := "9.8.7.6"
	rows := sqlmock.NewRows([]string{"aws_public_ip_assignment_public_ip",
		"aws_public_ip_assignment_hostname",
		"aws_resource_arn_id",
		"aws_resource_meta",
		"aws_region_region",
		"aws_resource_type_resource_type",
		"aws_account_account",
		"aws_account_id",
		"aws_account_account",
		"owner_login",
		"owner_email",
		"owner_name",
		"owner_valid",
		"champion_login",
		"champion_email",
		"champion_name",
		"champion_valid"}).AddRow("9.8.7.6",
		"google.com",
		"rid",
		[]byte("{\"hi\":\"there\"}"),
		"region",
		"type",
		"aid",
		"1",
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true).AddRow("8.7.6.5",
		"yahoo.com",
		"rid2",
		[]byte("{\"bye\":\"now\"}"),
		"region2",
		"type2",
		"aid2",
		"2",
		"aid2",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true)

	mock.ExpectQuery("select").WithArgs(ipAddress, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByIP(context.Background(), at, ipAddress)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	expected := []domain.CloudAssetDetails{
		domain.CloudAssetDetails{
			PublicIPAddresses: []string{"9.8.7.6"},
			Hostnames:         []string{"google.com"},
			ResourceType:      "type",
			AccountID:         "aid",
			Region:            "region",
			ARN:               "rid",
			Tags:              map[string]string{"hi": "there"},
			AccountOwner: domain.AccountOwner{
				AccountID: "aid",
				Owner: domain.Person{
					Login: "login",
					Email: "email@atlassian.com",
					Name:  "name",
					Valid: true,
				},
				Champions: []domain.Person{
					{
						Login: "login2",
						Email: "email2@atlassian.com",
						Name:  "name2",
						Valid: true,
					},
				},
			},
		},
		domain.CloudAssetDetails{
			PublicIPAddresses: []string{"8.7.6.5"},
			Hostnames:         []string{"yahoo.com"},
			ResourceType:      "type2",
			AccountID:         "aid2",
			Region:            "region2",
			ARN:               "rid2",
			Tags:              map[string]string{"bye": "now"},
			AccountOwner: domain.AccountOwner{
				AccountID: "aid2",
				Owner: domain.Person{
					Login: "login",
					Email: "email@atlassian.com",
					Name:  "name",
					Valid: true,
				},
				Champions: []domain.Person{
					{
						Login: "login2",
						Email: "email2@atlassian.com",
						Name:  "name2",
						Valid: true,
					},
				},
			},
		},
	}

	assertArrayEqualIgnoreOrder(t, expected, results)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetHostnamesAtTimeSchema1(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	withSchemaVersion(1, mock)
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

func TestGetHostnamesAtTimeSchema2(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	withSchemaVersion(4, mock)
	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	hostname := "yahoo.com"

	rows := sqlmock.NewRows([]string{"aws_public_ip_assignment_public_ip",
		"aws_public_ip_assignment_aws_hostname",
		"aws_resource_arn_id",
		"aws_resource_meta",
		"aws_region_region",
		"aws_resource_type_resource_type",
		"aws_account_account",
		"aws_account_id",
		"aws_account_account",
		"owner_login",
		"owner_email",
		"owner_name",
		"owner_valid",
		"champion_login",
		"champion_email",
		"champion_name",
		"champion_valid"}).AddRow("44.33.22.11",
		"yahoo.com",
		"rid",
		[]byte("{\"hi\":\"there3\"}"),
		"region",
		"type",
		"aid",
		"1",
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true)

	mock.ExpectQuery("select").WithArgs(hostname, at).WillReturnRows(rows).RowsWillBeClosed()

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
		AccountOwner: domain.AccountOwner{
			AccountID: "aid",
			Owner: domain.Person{
				Login: "login",
				Email: "email@atlassian.com",
				Name:  "name",
				Valid: true,
			},
			Champions: []domain.Person{
				{
					Login: "login2",
					Email: "email2@atlassian.com",
					Name:  "name2",
					Valid: true,
				},
			},
		},
	}, results[0])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetHostnamesAtTimeMultiRowsSchema1(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	withSchemaVersion(1, mock)
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

func TestGetHostnamesAtTimeMultiRowsSchema2(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	withSchemaVersion(4, mock)
	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	hostname := "yahoo.com"

	rows := sqlmock.NewRows([]string{"aws_public_ip_assignment_public_ip",
		"aws_public_ip_assignment_aws_hostname",
		"aws_resource_arn_id",
		"aws_resource_meta",
		"aws_region_region",
		"aws_resource_type_resource_type",
		"aws_account_account",
		"aws_account_id",
		"aws_account_account",
		"owner_login",
		"owner_email",
		"owner_name",
		"owner_valid",
		"champion_login",
		"champion_email",
		"champion_name",
		"champion_valid"}).AddRow("44.33.22.11",
		"yahoo.com",
		"rid",
		[]byte("{\"hi\":\"there3\"}"),
		"region",
		"type",
		"aid",
		"1",
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true).AddRow("9.8.7.6",
		"yahoo.com",
		"rid",
		[]byte("{\"hi\":\"there4\"}"),
		"region",
		"type",
		"aid",
		"2",
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true).AddRow("9.8.7.6",
		nil,
		"rid",
		[]byte("{\"hi\":\"there5\"}"),
		"region",
		"type",
		"aid",
		"3",
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true)

	mock.ExpectQuery("select").WithArgs(hostname, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByHostname(context.Background(), at, hostname)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 1, len(results))
	assert.Equal(t, domain.CloudAssetDetails{
		PrivateIPAddresses: []string(nil),
		PublicIPAddresses:  []string{"44.33.22.11", "9.8.7.6"},
		Hostnames:          []string{"yahoo.com"},
		ResourceType:       "type",
		AccountID:          "aid",
		Region:             "region",
		ARN:                "rid",
		Tags:               map[string]string{"hi": "there3"},
		AccountOwner: domain.AccountOwner{
			AccountID: "aid",
			Owner: domain.Person{
				Login: "login",
				Email: "email@atlassian.com",
				Name:  "name",
				Valid: true,
			},
			Champions: []domain.Person{
				{
					Login: "login2",
					Email: "email2@atlassian.com",
					Name:  "name2",
					Valid: true,
				},
			},
		},
	}, results[0])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetByResourceIDEmpty(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	withSchemaVersion(4, mock)
	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	const resID = "resid"
	rows := sqlmock.NewRows([]string{"aws_private_ip_assignment_private_ip",
		"aws_public_ip_assignment_public_ip",
		"aws_public_ip_assignment_aws_hostname",
		"aws_resource_type_resource_type",
		"aws_account_account",
		"aws_region_region",
		"aws_resource_meta",
		"aws_resource_aws_account_id",
	})

	mock.ExpectQuery("select").WithArgs(resID, at).WillReturnRows(rows).RowsWillBeClosed()
	results, err := thedb.FetchByResourceID(context.Background(), at, resID)
	if err != nil {
		t.Errorf("error was not expected during lookup: %s", err)
	}
	assert.Equal(t, 0, len(results))
}

func TestGetResourceIDAtTime(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	withSchemaVersion(4, mock)
	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	const resID = "resid"

	rows := sqlmock.NewRows([]string{"aws_private_ip_assignment_private_ip",
		"aws_public_ip_assignment_public_ip",
		"aws_public_ip_assignment_aws_hostname",
		"aws_resource_type_resource_type",
		"aws_account_account",
		"aws_region_region",
		"aws_resource_meta",
		"aws_resource_aws_account_id",
		"aws_account_account",
		"owner_login",
		"owner_email",
		"owner_name",
		"owner_valid",
		"champion_login",
		"champion_email",
		"champion_name",
		"champion_valid",
	}).AddRow("172.16.3.3",
		"44.33.22.11",
		"yahoo.com",
		"type",
		"aid",
		"region",
		[]byte("{\"hi\":\"there3\"}"),
		1,
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true)

	mock.ExpectQuery("select").WithArgs(resID, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByResourceID(context.Background(), at, resID)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assert.Equal(t, 1, len(results))
	assert.Equal(t, domain.CloudAssetDetails{
		PrivateIPAddresses: []string{"172.16.3.3"},
		PublicIPAddresses:  []string{"44.33.22.11"},
		Hostnames:          []string{"yahoo.com"},
		ResourceType:       "type",
		AccountID:          "aid",
		Region:             "region",
		ARN:                "resid",
		Tags:               map[string]string{"hi": "there3"},
		AccountOwner: domain.AccountOwner{
			AccountID: "aid",
			Owner: domain.Person{
				Login: "login",
				Email: "email@atlassian.com",
				Name:  "name",
				Valid: true,
			},
			Champions: []domain.Person{
				{
					Login: "login2",
					Email: "email2@atlassian.com",
					Name:  "name2",
					Valid: true,
				},
			},
		},
	}, results[0])

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetResourceIDAtTimeMoreThanOnePublicIPs(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	withSchemaVersion(4, mock)

	rows := sqlmock.NewRows([]string{"aws_private_ip_assignment_private_ip",
		"aws_public_ip_assignment_public_ip",
		"aws_public_ip_assignment_aws_hostname",
		"aws_resource_type_resource_type",
		"aws_account_account",
		"aws_region_region",
		"aws_resource_meta",
		"aws_resource_aws_account_id",
		"aws_account_account",
		"owner_login",
		"owner_email",
		"owner_name",
		"owner_valid",
		"champion_login",
		"champion_email",
		"champion_name",
		"champion_valid",
	}).AddRow("172.16.3.3",
		"44.33.22.11",
		"yahoo.com",
		"type",
		"aid",
		"region",
		[]byte("{\"hi\":\"there3\"}"),
		1,
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true).AddRow("172.16.3.3",
		"9.8.7.6",
		"yahoo.com",
		"type",
		"aid",
		"region",
		[]byte("{\"hi\":\"there3\"}"),
		1,
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true)

	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	resID := "resid"
	mock.ExpectQuery("select").WithArgs(resID, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByResourceID(context.Background(), at, resID)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	assertArrayEqualIgnoreOrder(t, []domain.CloudAssetDetails{
		{
			PrivateIPAddresses: []string{"172.16.3.3"},
			PublicIPAddresses:  []string{"44.33.22.11", "9.8.7.6"},
			Hostnames:          []string{"yahoo.com"},
			ResourceType:       "type",
			AccountID:          "aid",
			Region:             "region",
			ARN:                "resid",
			Tags:               map[string]string{"hi": "there3"},
			AccountOwner: domain.AccountOwner{
				AccountID: "aid",
				Owner: domain.Person{
					Name:  "name",
					Login: "login",
					Email: "email@atlassian.com",
					Valid: true,
				},
				Champions: []domain.Person{
					{
						Name:  "name2",
						Login: "login2",
						Email: "email2@atlassian.com",
						Valid: true,
					},
				},
			},
		},
	}, results)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetResourceIDAtTimeMoreThanOneChampions(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	withSchemaVersion(4, mock)

	rows := sqlmock.NewRows([]string{"aws_private_ip_assignment_private_ip",
		"aws_public_ip_assignment_public_ip",
		"aws_public_ip_assignment_aws_hostname",
		"aws_resource_type_resource_type",
		"aws_account_account",
		"aws_region_region",
		"aws_resource_meta",
		"aws_resource_aws_account_id",
		"aws_account_account",
		"owner_login",
		"owner_email",
		"owner_name",
		"owner_valid",
		"champion_login",
		"champion_email",
		"champion_name",
		"champion_valid",
	}).AddRow("172.16.3.3",
		"44.33.22.11",
		"yahoo.com",
		"type",
		"aid",
		"region",
		[]byte("{\"hi\":\"there3\"}"),
		1,
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login2",
		"email2@atlassian.com",
		"name2",
		true).AddRow("172.16.3.3",
		"44.33.22.11",
		"yahoo.com",
		"type",
		"aid",
		"region",
		[]byte("{\"hi\":\"there3\"}"),
		1,
		"aid",
		"login",
		"email@atlassian.com",
		"name",
		true,
		"login3",
		"email3@atlassian.com",
		"name3",
		true)

	at, _ := time.Parse(time.RFC3339, "2019-04-09T08:55:35+00:00")
	resID := "resid"
	mock.ExpectQuery("select").WithArgs(resID, at).WillReturnRows(rows).RowsWillBeClosed()

	results, err := thedb.FetchByResourceID(context.Background(), at, resID)
	if err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	actual := results[0]

	sort.SliceStable(actual.AccountOwner.Champions, func(i, j int) bool {
		return actual.AccountOwner.Champions[i].Name < actual.AccountOwner.Champions[j].Name
	})

	assert.Equal(t, 1, len(results))
	assert.Equal(t, domain.CloudAssetDetails{
		PrivateIPAddresses: []string{"172.16.3.3"},
		PublicIPAddresses:  []string{"44.33.22.11"},
		Hostnames:          []string{"yahoo.com"},
		ResourceType:       "type",
		AccountID:          "aid",
		Region:             "region",
		ARN:                "resid",
		Tags:               map[string]string{"hi": "there3"},
		AccountOwner: domain.AccountOwner{
			AccountID: "aid",
			Owner: domain.Person{
				Name:  "name",
				Login: "login",
				Email: "email@atlassian.com",
				Valid: true,
			},
			Champions: []domain.Person{
				{
					Name:  "name2",
					Login: "login2",
					Email: "email2@atlassian.com",
					Valid: true,
				},
				{
					Name:  "name3",
					Login: "login3",
					Email: "email3@atlassian.com",
					Valid: true,
				},
			},
		},
	}, actual)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
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
		sqldb: mockdb,
		now:   func() time.Time { return createdAt },
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
		sqldb: mockdb,
		now:   func() time.Time { return createdAt },
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
		sqldb: mockdb,
		now:   func() time.Time { return createdAt },
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
		sqldb: mockdb,
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
		sqldb: mockdb,
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
		sqldb: mockdb,
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
		sqldb: mockdb,
		now:   func() time.Time { return createdAt },
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
		sqldb: mockdb,
		now:   func() time.Time { return createdAt },
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
		sqldb: mockdb,
		now:   func() time.Time { return createdAt },
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
		sqldb: mockdb,
		now:   func() time.Time { return createdAt },
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
		sqldb: mockdb,
		now:   func() time.Time { return createdAt },
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
		sqldb: mockdb,
		now:   func() time.Time { return createdAt },
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
		sqldb: mockdb,
		now:   func() time.Time { return createdAt },
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
		sqldb: mockdb,
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
		sqldb: mockdb,
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
		sqldb: mockdb,
		now:   func() time.Time { return now },
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
		sqldb: mockdb,
		now:   func() time.Time { return now },
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
		sqldb: mockdb,
		now:   func() time.Time { return now },
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
		sqldb: mockdb,
		now:   func() time.Time { return now },
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
		sqldb: mockdb,
		now:   func() time.Time { return now },
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
		sqldb: mockdb,
		now:   func() time.Time { return now },
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
	return fakeCloudChange("ADDED")
}
func fakeCloudChange(changeType string) domain.CloudAssetChanges {
	privateIPs := []string{"4.3.2.1"}
	publicIPs := []string{"8.7.6.5"}
	hostnames := []string{"google.com"}
	networkChangesArray := []domain.NetworkChanges{
		domain.NetworkChanges{
			PrivateIPAddresses: privateIPs,
			PublicIPAddresses:  publicIPs,
			Hostnames:          hostnames,
			ChangeType:         changeType,
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
			expectedValCopy := expectedVal
			actualValCopy := actualVal
			if compareArray(expectedVal.PublicIPAddresses, actualVal.PublicIPAddresses) &&
				compareArray(expectedVal.PrivateIPAddresses, actualVal.PrivateIPAddresses) &&
				compareArray(expectedVal.Hostnames, actualVal.Hostnames) {
				expectedValCopy.PublicIPAddresses = []string{}
				expectedValCopy.PrivateIPAddresses = []string{}
				expectedValCopy.PrivateIPAddresses = []string{}
				actualValCopy.PublicIPAddresses = []string{}
				actualValCopy.PrivateIPAddresses = []string{}
				actualValCopy.PrivateIPAddresses = []string{}
			}
			e, _ := json.Marshal(expectedValCopy)
			a, _ := json.Marshal(actualValCopy)

			// likely due to timestamp, DeepEqual(expectedValCopy, actualValCopy) would not work, so checking the Marshaled JSON:
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

func compareArray(expected, actual []string) bool {
	// brute force
	if len(expected) != len(actual) {
		return false
	}
	equalityCount := 0
	for _, expectedVal := range expected {
		for _, actualVal := range actual {
			if expectedVal == actualVal {
				equalityCount++
				break
			}
		}
	}
	return len(expected) == equalityCount
}

func TestStoreV2ErrorEnsureResource(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	theDB := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin()
	mock.ExpectExec("with sel as").WithArgs("arn", "region", "aid", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnError(errors.New("failed to store resource"))
	mock.ExpectRollback()

	ctx := context.Background()

	if err = theDB.StoreV2(ctx, fakeCloudAssetChanges()); err == nil {
		t.Errorf("error was expected while saving resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestStoreV2Assign(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	theDB := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin()
	mock.ExpectExec("with sel as").WithArgs("arn", "region", "aid", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnResult(sqlmock.NewResult(1, 1))
	timestamp, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	// NB we need to escape '$' and other special chars as the value passed as expected query is a regexp
	mock.ExpectExec(regexp.QuoteMeta(`update aws_private_ip_assignment`)).WithArgs(timestamp, "4.3.2.1", "arn").WillReturnResult(sqlmock.NewResult(1, 1))              // nolint
	mock.ExpectExec(regexp.QuoteMeta(`update aws_public_ip_assignment`)).WithArgs(timestamp, "8.7.6.5", "arn", "google.com").WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectCommit()

	ctx := context.Background()

	if err = theDB.StoreV2(ctx, fakeCloudAssetChanges()); err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestStoreV2Remove(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	theDB := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin()
	mock.ExpectExec("with sel as").WithArgs("arn", "region", "aid", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnResult(sqlmock.NewResult(1, 1))
	timestamp, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	// NB we need to escape '$' and other special chars as the value passed as expected query is a regexp
	mock.ExpectExec(regexp.QuoteMeta(`update aws_private_ip_assignment`)).WithArgs(timestamp, "4.3.2.1", "arn").WillReturnResult(sqlmock.NewResult(1, 1))              // nolint
	mock.ExpectExec(regexp.QuoteMeta(`update aws_public_ip_assignment`)).WithArgs(timestamp, "8.7.6.5", "arn", "google.com").WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectCommit()

	ctx := context.Background()

	if err = theDB.StoreV2(ctx, fakeCloudChange("DELETED")); err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestStoreV2FailPrivate(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	theDB := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin()
	mock.ExpectExec("with sel as").WithArgs("arn", "region", "aid", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnResult(sqlmock.NewResult(1, 1))
	timestamp, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	// NB we need to escape '$' and other special chars as the value passed as expected query is a regexp
	mock.ExpectExec(regexp.QuoteMeta(`update aws_private_ip_assignment`)).WithArgs(timestamp, "4.3.2.1", "arn").WillReturnError(errors.New("failed to store assignment"))
	mock.ExpectRollback()

	ctx := context.Background()

	if err = theDB.StoreV2(ctx, fakeCloudChange("DELETED")); err == nil {
		t.Errorf("error was expected while saving private resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestStoreV2FailPublic(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	theDB := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin()
	mock.ExpectExec("with sel as").WithArgs("arn", "region", "aid", "rtype", []byte("{\"tag1\":\"val1\"}")).WillReturnResult(sqlmock.NewResult(1, 1))
	timestamp, _ := time.Parse(time.RFC3339, "2019-04-09T08:29:35+00:00")
	// NB we need to escape '$' and other special chars as the value passed as expected query is a regexp
	mock.ExpectExec(regexp.QuoteMeta(`update aws_private_ip_assignment`)).WithArgs(timestamp, "4.3.2.1", "arn").WillReturnResult(sqlmock.NewResult(1, 1))                              // nolint
	mock.ExpectExec(regexp.QuoteMeta(`update aws_public_ip_assignment`)).WithArgs(timestamp, "8.7.6.5", "arn", "google.com").WillReturnError(errors.New("failed to store assignment")) // nolint
	mock.ExpectRollback()

	ctx := context.Background()

	if err = theDB.StoreV2(ctx, fakeCloudChange("DELETED")); err == nil {
		t.Errorf("error was expected while saving private resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestStoreV2FailTxOpen(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	theDB := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin().WillReturnError(errors.New("failed to start transaction"))
	ctx := context.Background()
	if err = theDB.StoreV2(ctx, fakeCloudChange("DELETED")); err == nil {
		t.Errorf("error was expected while saving resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestBackFillEventsLocallySelectErr(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	theDB := DB{
		sqldb: mockdb,
	}
	from := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC)
	to := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`
select ae.ts,
       ae.aws_resources_id,
       ar.type,
       ar.region,
       ar.account_id,
       ar.meta,
       ae.aws_ips_ip,
       ae.aws_hostnames_hostname,
       ae.is_join,
       ae.is_public
from aws_events_ips_hostnames as ae
         left join aws_resources ar on ae.aws_resources_id = ar.id
`).WillReturnError(errors.New("can not query"))
	ctx := context.Background()
	if err := theDB.BackFillEventsLocally(ctx, from, to); err == nil {
		t.Errorf("error expected when running BackFillEventsLocally")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestBackFillEventsLocally(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	theDB := DB{
		sqldb: mockdb,
	}
	from := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC)
	to := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"ae.ts", "ae.aws_resources_id", "ar.type", "ar.region", "ar.account_id", "ar.meta", "ae.aws_ips_ip",
		"ae.aws_hostnames_hostname", "ae.is_join", "ae.is_public",
	}).AddRow(
		from, "arnid", "type", "region", "account", "{key: value}", "8.8.8.8",
		"hostname", true, true) // public assign
	rows.AddRow(
		to, "arnid", "type", "region", "account", "{key: value}", "8.8.8.8",
		"hostname", false, true) // public release
	rows.AddRow(
		from, "arnid", "type", "region", "account", "{key: value}", "10.10.10.10",
		sql.NullString{Valid: false}, true, false) //private assign
	rows.AddRow(
		to, "arnid", "type", "region", "account", "{key: value}", "10.10.10.10",
		sql.NullString{Valid: false}, false, false) //private release
	mock.ExpectQuery(`
select ae.ts,
       ae.aws_resources_id,
       ar.type,
       ar.region,
       ar.account_id,
       ar.meta,
       ae.aws_ips_ip,
       ae.aws_hostnames_hostname,
       ae.is_join,
       ae.is_public
from aws_events_ips_hostnames as ae
         left join aws_resources ar on ae.aws_resources_id = ar.id
`).WillReturnRows(rows)
	//public
	//assign
	mock.ExpectBegin()
	mock.ExpectExec("with sel as").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`update aws_public_ip_assignment`)).WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectCommit()
	//release
	mock.ExpectBegin()
	mock.ExpectExec("with sel as").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`update aws_public_ip_assignment`)).WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectCommit()
	//private
	//assign
	mock.ExpectBegin()
	mock.ExpectExec("with sel as").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`update aws_private_ip_assignment`)).WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectCommit()
	//release
	mock.ExpectBegin()
	mock.ExpectExec("with sel as").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`update aws_private_ip_assignment`)).WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectCommit()

	ctx := context.Background()

	if err := theDB.BackFillEventsLocally(ctx, from, to); err != nil {
		t.Errorf("no error expected when running BackFillEventsLocally")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func fakeAccountOwnerInput() domain.AccountOwner {
	return domain.AccountOwner{
		AccountID: "awsaccountid123",
		Owner: domain.Person{
			Name:  "john dane",
			Login: "jdane",
			Email: "jdane@atlassian.com",
			Valid: true,
		},
		Champions: []domain.Person{
			{
				Name:  "john dane",
				Login: "jdane",
				Email: "jdane@atlassian.com",
				Valid: true,
			},
		},
	}
}

func fakeAccountOwnerInputNoChampion() domain.AccountOwner {
	return domain.AccountOwner{
		AccountID: "awsaccountid123",
		Owner: domain.Person{
			Name:  "john dane",
			Login: "jdane",
			Email: "jdane@atlassian.com",
			Valid: true,
		},
	}
}

func TestAccountOwnerTxBeginFailure(t *testing.T) {
	// no panics, in other words
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin().WillReturnError(fmt.Errorf("could not start transaction"))

	ctx := context.Background()

	if err = thedb.StoreAccountOwner(ctx, fakeAccountOwnerInput()); err == nil {
		t.Errorf("was expecting an error, but there was none")
	}
	assert.Equal(t, "could not start transaction", err.Error())

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestShouldRollbackOnFailureToINSERTAccountOwner(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("awsaccountid123").WillReturnError(fmt.Errorf("some error"))
	mock.ExpectRollback()

	ctx := context.Background()

	if err = thedb.StoreAccountOwner(ctx, fakeAccountOwnerInput()); err == nil {
		t.Errorf("was expecting an error, but there was none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestShouldRollbackOnFailureToINSERTAccountOwner2(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("awsaccountid123").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO").WithArgs("jdane", "jdane@atlassian.com", "john dane", true).WillReturnError(fmt.Errorf("some error"))
	mock.ExpectRollback()

	ctx := context.Background()

	if err = thedb.StoreAccountOwner(ctx, fakeAccountOwnerInput()); err == nil {
		t.Errorf("was expecting an error, but there was none")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestShouldINSERTAccountOwnerWithChampion(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("awsaccountid123").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO").WithArgs("jdane", "jdane@atlassian.com", "john dane", true).WillReturnResult(sqlmock.NewResult(1, 1))
	row := sqlmock.NewRows([]string{
		"id",
	}).AddRow('1')
	mock.ExpectQuery("SELECT").WithArgs("jdane").WillReturnRows(row)
	row2 := sqlmock.NewRows([]string{
		"id",
	}).AddRow('1')
	mock.ExpectQuery("SELECT").WithArgs("awsaccountid123").WillReturnRows(row2)
	mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO").WithArgs("jdane", "jdane@atlassian.com", "john dane", true).WillReturnResult(sqlmock.NewResult(1, 1))
	row3 := sqlmock.NewRows([]string{
		"id",
	}).AddRow('1')
	mock.ExpectQuery("SELECT").WithArgs("jdane").WillReturnRows(row3)
	mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()

	if err = thedb.StoreAccountOwner(ctx, fakeAccountOwnerInput()); err != nil {
		t.Errorf("did not expect error while inserting resources")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestShouldINSERTAccountOwnerNoChampion(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("awsaccountid123").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO").WithArgs("jdane", "jdane@atlassian.com", "john dane", true).WillReturnResult(sqlmock.NewResult(1, 1))
	row := sqlmock.NewRows([]string{
		"id",
	}).AddRow('1')
	mock.ExpectQuery("SELECT").WithArgs("jdane").WillReturnRows(row)
	row2 := sqlmock.NewRows([]string{
		"id",
	}).AddRow('1')
	mock.ExpectQuery("SELECT").WithArgs("awsaccountid123").WillReturnRows(row2)
	mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	ctx := context.Background()

	if err = thedb.StoreAccountOwner(ctx, fakeAccountOwnerInputNoChampion()); err != nil {
		t.Errorf("did not expect error while inserting resources")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGoldenPathINSERTAccountOwner(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO").WithArgs("awsaccountid123").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO").WithArgs("jdane", "jdane@atlassian.com", "john dane", true).WillReturnResult(sqlmock.NewResult(1, 1))
	row := sqlmock.NewRows([]string{
		"id",
	}).AddRow('1')
	mock.ExpectQuery("SELECT").WithArgs("jdane").WillReturnRows(row)
	row2 := sqlmock.NewRows([]string{
		"id",
	}).AddRow('1')
	mock.ExpectQuery("SELECT").WithArgs("awsaccountid123").WillReturnRows(row2)
	mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO").WithArgs("jdane", "jdane@atlassian.com", "john dane", true).WillReturnResult(sqlmock.NewResult(1, 1))
	row3 := sqlmock.NewRows([]string{
		"id",
	}).AddRow('1')
	mock.ExpectQuery("SELECT").WithArgs("jdane").WillReturnRows(row3)
	mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	fakeContext, _ := mockdb.BeginTx(context.Background(), nil)

	if err = thedb.storeAccountOwner(ctx, fakeAccountOwnerInput(), fakeContext); err != nil {
		t.Errorf("error was not expected while saving resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

}

func TestResIDFromARN(t *testing.T) {
	testCases := []struct {
		Name     string
		Arn      string
		Expected string
	}{
		{"TestResIDFromARNForENI",
			"arn:aws:ec2:us-west-2:909420000000:network-interface/eni-049a0265f0663b9ac",
			"eni-049a0265f0663b9ac"},
		{"TestResIDFromARNForEC2",
			"arn:aws:ec2:us-west-2:909420000000:instance/i-0bd0340bdada89d2f",
			"i-0bd0340bdada89d2f"},
		{"TestResIDFromARNForALB",
			"arn:aws:ec2:us-west-2:909420000000:loadbalancer/app/my-sec-dev-one-alb/2b9ae31f54b6fa76",
			"app/my-sec-dev-one-alb/2b9ae31f54b6fa76"},
		{"TestResIDFromARNForCLB",
			"arn:aws:ec2:us-west-2:909420000000:loadbalancer/my-classic-lb",
			"my-classic-lb"},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			actual := resIDFromARN(tc.Arn)
			if actual != tc.Expected {
				t.Errorf("%s != %s", actual, tc.Expected)
			} else {
				t.Logf("%s == %s", actual, tc.Expected)
			}
		})
	}
}

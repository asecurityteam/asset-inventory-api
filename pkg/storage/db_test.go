package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

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

func withSchemaVersion(version uint, mock sqlmock.Sqlmock) {
	rows := sqlmock.NewRows([]string{"version"}).AddRow(version)
	mock.ExpectQuery("select version from schema_migrations").WillReturnRows(rows).RowsWillBeClosed()
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
			AccountID: toStringPointer("aid"),
			Owner: domain.Person{
				Login: toStringPointer("login"),
				Email: toStringPointer("email@atlassian.com"),
				Name:  toStringPointer("name"),
				Valid: toBoolPointer(true),
			},
			Champions: []domain.Person{
				{
					Login: toStringPointer("login2"),
					Email: toStringPointer("email2@atlassian.com"),
					Name:  toStringPointer("name2"),
					Valid: toBoolPointer(true),
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
			AccountID: toStringPointer("aid"),
			Owner: domain.Person{
				Login: toStringPointer("login"),
				Email: toStringPointer("email@atlassian.com"),
				Name:  toStringPointer("name"),
				Valid: toBoolPointer(true),
			},
			Champions: []domain.Person{
				{
					Login: toStringPointer("login2"),
					Email: toStringPointer("email2@atlassian.com"),
					Name:  toStringPointer("name2"),
					Valid: toBoolPointer(true),
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
		{
			PrivateIPAddresses: []string{"172.16.2.2"},
			ResourceType:       "type",
			AccountID:          "aid",
			Region:             "region",
			ARN:                "rid",
			Tags:               map[string]string{"hi": "there1"},
			AccountOwner: domain.AccountOwner{
				AccountID: toStringPointer("aid"),
				Owner: domain.Person{
					Login: toStringPointer("login"),
					Email: toStringPointer("email@atlassian.com"),
					Name:  toStringPointer("name"),
					Valid: toBoolPointer(true),
				},
				Champions: []domain.Person{
					{
						Login: toStringPointer("login2"),
						Email: toStringPointer("email2@atlassian.com"),
						Name:  toStringPointer("name2"),
						Valid: toBoolPointer(true),
					},
				},
			},
		},
		{
			PrivateIPAddresses: []string{"172.16.3.3"},
			ResourceType:       "type2",
			AccountID:          "aid2",
			Region:             "region2",
			ARN:                "rid2",
			Tags:               map[string]string{"bye": "now"},
			AccountOwner: domain.AccountOwner{
				AccountID: toStringPointer("aid2"),
				Owner: domain.Person{
					Login: toStringPointer("login"),
					Email: toStringPointer("email@atlassian.com"),
					Name:  toStringPointer("name"),
					Valid: toBoolPointer(true),
				},
				Champions: []domain.Person{
					{
						Login: toStringPointer("login2"),
						Email: toStringPointer("email2@atlassian.com"),
						Name:  toStringPointer("name2"),
						Valid: toBoolPointer(true),
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
		{
			PublicIPAddresses: []string{"9.8.7.6"},
			Hostnames:         []string{"google.com"},
			ResourceType:      "type",
			AccountID:         "aid",
			Region:            "region",
			ARN:               "rid",
			Tags:              map[string]string{"hi": "there"},
			AccountOwner: domain.AccountOwner{
				AccountID: toStringPointer("aid"),
				Owner: domain.Person{
					Login: toStringPointer("login"),
					Email: toStringPointer("email@atlassian.com"),
					Name:  toStringPointer("name"),
					Valid: toBoolPointer(true),
				},
				Champions: []domain.Person{
					{
						Login: toStringPointer("login2"),
						Email: toStringPointer("email2@atlassian.com"),
						Name:  toStringPointer("name2"),
						Valid: toBoolPointer(true),
					},
				},
			},
		},
		{
			PublicIPAddresses: []string{"8.7.6.5"},
			Hostnames:         []string{"yahoo.com"},
			ResourceType:      "type2",
			AccountID:         "aid2",
			Region:            "region2",
			ARN:               "rid2",
			Tags:              map[string]string{"bye": "now"},
			AccountOwner: domain.AccountOwner{
				AccountID: toStringPointer("aid2"),
				Owner: domain.Person{
					Login: toStringPointer("login"),
					Email: toStringPointer("email@atlassian.com"),
					Name:  toStringPointer("name"),
					Valid: toBoolPointer(true),
				},
				Champions: []domain.Person{
					{
						Login: toStringPointer("login2"),
						Email: toStringPointer("email2@atlassian.com"),
						Name:  toStringPointer("name2"),
						Valid: toBoolPointer(true),
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

func TestGetHostnamesAtTimeSchema2(t *testing.T) {
	mockdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockdb.Close()

	thedb := DB{
		sqldb: mockdb,
	}

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
			AccountID: toStringPointer("aid"),
			Owner: domain.Person{
				Login: toStringPointer("login"),
				Email: toStringPointer("email@atlassian.com"),
				Name:  toStringPointer("name"),
				Valid: toBoolPointer(true),
			},
			Champions: []domain.Person{
				{
					Login: toStringPointer("login2"),
					Email: toStringPointer("email2@atlassian.com"),
					Name:  toStringPointer("name2"),
					Valid: toBoolPointer(true),
				},
			},
		},
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
			AccountID: toStringPointer("aid"),
			Owner: domain.Person{
				Login: toStringPointer("login"),
				Email: toStringPointer("email@atlassian.com"),
				Name:  toStringPointer("name"),
				Valid: toBoolPointer(true),
			},
			Champions: []domain.Person{
				{
					Login: toStringPointer("login2"),
					Email: toStringPointer("email2@atlassian.com"),
					Name:  toStringPointer("name2"),
					Valid: toBoolPointer(true),
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
			AccountID: toStringPointer("aid"),
			Owner: domain.Person{
				Login: toStringPointer("login"),
				Email: toStringPointer("email@atlassian.com"),
				Name:  toStringPointer("name"),
				Valid: toBoolPointer(true),
			},
			Champions: []domain.Person{
				{
					Login: toStringPointer("login2"),
					Email: toStringPointer("email2@atlassian.com"),
					Name:  toStringPointer("name2"),
					Valid: toBoolPointer(true),
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
				AccountID: toStringPointer("aid"),
				Owner: domain.Person{
					Name:  toStringPointer("name"),
					Login: toStringPointer("login"),
					Email: toStringPointer("email@atlassian.com"),
					Valid: toBoolPointer(true),
				},
				Champions: []domain.Person{
					{
						Name:  toStringPointer("name2"),
						Login: toStringPointer("login2"),
						Email: toStringPointer("email2@atlassian.com"),
						Valid: toBoolPointer(true),
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
		return *actual.AccountOwner.Champions[i].Name < *actual.AccountOwner.Champions[j].Name
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
			AccountID: toStringPointer("aid"),
			Owner: domain.Person{
				Name:  toStringPointer("name"),
				Login: toStringPointer("login"),
				Email: toStringPointer("email@atlassian.com"),
				Valid: toBoolPointer(true),
			},
			Champions: []domain.Person{
				{
					Name:  toStringPointer("name2"),
					Login: toStringPointer("login2"),
					Email: toStringPointer("email2@atlassian.com"),
					Valid: toBoolPointer(true),
				},
				{
					Name:  toStringPointer("name3"),
					Login: toStringPointer("login3"),
					Email: toStringPointer("email3@atlassian.com"),
					Valid: toBoolPointer(true),
				},
			},
		},
	}, actual)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func fakeCloudAssetChanges() domain.CloudAssetChanges {
	return fakeCloudChange("ADDED")
}
func fakeCloudChange(changeType string) domain.CloudAssetChanges {
	privateIPs := []string{"4.3.2.1"}
	publicIPs := []string{"8.7.6.5"}
	hostnames := []string{"google.com"}
	relatedResources := []string{"app/marketp-ALB-eeeeeee5555555/ffffffff66666666"}
	networkChangesArray := []domain.NetworkChanges{
		{
			PrivateIPAddresses: privateIPs,
			PublicIPAddresses:  publicIPs,
			Hostnames:          hostnames,
			RelatedResources:   relatedResources,
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

	if err = theDB.Store(ctx, fakeCloudAssetChanges()); err == nil {
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
	mock.ExpectExec(regexp.QuoteMeta(`update aws_private_ip_assignment`)).WithArgs(timestamp, "4.3.2.1", "arn").WillReturnResult(sqlmock.NewResult(1, 1))                                         // nolint
	mock.ExpectExec(regexp.QuoteMeta(`update aws_public_ip_assignment`)).WithArgs(timestamp, "8.7.6.5", "arn", "google.com").WillReturnResult(sqlmock.NewResult(1, 1))                            // nolint
	mock.ExpectExec(regexp.QuoteMeta(`update aws_resource_relationship`)).WithArgs(timestamp, "app/marketp-ALB-eeeeeee5555555/ffffffff66666666", "arn").WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectCommit()

	ctx := context.Background()

	if err = theDB.Store(ctx, fakeCloudAssetChanges()); err != nil {
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
	mock.ExpectExec(regexp.QuoteMeta(`update aws_private_ip_assignment`)).WithArgs(timestamp, "4.3.2.1", "arn").WillReturnResult(sqlmock.NewResult(1, 1))                                         // nolint
	mock.ExpectExec(regexp.QuoteMeta(`update aws_public_ip_assignment`)).WithArgs(timestamp, "8.7.6.5", "arn", "google.com").WillReturnResult(sqlmock.NewResult(1, 1))                            // nolint
	mock.ExpectExec(regexp.QuoteMeta(`update aws_resource_relationship`)).WithArgs(timestamp, "app/marketp-ALB-eeeeeee5555555/ffffffff66666666", "arn").WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectCommit()

	ctx := context.Background()

	if err = theDB.Store(ctx, fakeCloudChange("DELETED")); err != nil {
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

	if err = theDB.Store(ctx, fakeCloudChange("DELETED")); err == nil {
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

	if err = theDB.Store(ctx, fakeCloudChange("DELETED")); err == nil {
		t.Errorf("error was expected while saving private resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestStoreV2FailResourceRelationship(t *testing.T) {
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
	// Note: All related changes must be successful otherwise the whole transaction is canceled
	mock.ExpectExec(regexp.QuoteMeta(`update aws_private_ip_assignment`)).WithArgs(timestamp, "4.3.2.1", "arn").WillReturnResult(sqlmock.NewResult(1, 1))              // nolint
	mock.ExpectExec(regexp.QuoteMeta(`update aws_public_ip_assignment`)).WithArgs(timestamp, "8.7.6.5", "arn", "google.com").WillReturnResult(sqlmock.NewResult(1, 1)) // nolint
	mock.ExpectExec(regexp.QuoteMeta(`update aws_resource_relationship`)).WithArgs(timestamp, "app/marketp-ALB-eeeeeee5555555/ffffffff66666666", "arn").WillReturnError(errors.New("failed to store relationship"))
	mock.ExpectRollback()

	ctx := context.Background()

	if err = theDB.Store(ctx, fakeCloudChange("DELETED")); err == nil {
		t.Errorf("error was expected while saving resource relationship: %s", err)
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
	if err = theDB.Store(ctx, fakeCloudChange("DELETED")); err == nil {
		t.Errorf("error was expected while saving resource: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func fakeAccountOwnerInput() domain.AccountOwner {
	return domain.AccountOwner{
		AccountID: toStringPointer("awsaccountid123"),
		Owner: domain.Person{
			Name:  toStringPointer("john dane"),
			Login: toStringPointer("jdane"),
			Email: toStringPointer("jdane@atlassian.com"),
			Valid: toBoolPointer(true),
		},
		Champions: []domain.Person{
			{
				Name:  toStringPointer("john dane"),
				Login: toStringPointer("jdane"),
				Email: toStringPointer("jdane@atlassian.com"),
				Valid: toBoolPointer(true),
			},
		},
	}
}

func fakeAccountOwnerInputNoChampion() domain.AccountOwner {
	return domain.AccountOwner{
		AccountID: toStringPointer("awsaccountid123"),
		Owner: domain.Person{
			Name:  toStringPointer("john dane"),
			Login: toStringPointer("jdane"),
			Email: toStringPointer("jdane@atlassian.com"),
			Valid: toBoolPointer(true),
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
			assert.Equal(t, tc.Expected, actual, "Resource ID doesn't match expected output")
		})
	}
}

// Helper function to convert strings to pointers (for nullability)
func toStringPointer(s string) *string {
	return &s
}

// Helper function to convert booleans to pointers (for nullability)
func toBoolPointer(b bool) *bool {
	return &b
}

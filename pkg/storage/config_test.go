package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	postgresConfig := PostgresConfig{
		Hostname:     "localhost",
		Port:         99,
		Username:     "me!",
		Password:     "mypassword!",
		DatabaseName: "name",
		PartitionTTL: 365,
	}
	assert.Equal(t, "Postgres", postgresConfig.Name())
}

func TestShouldReturnSame(t *testing.T) {
	postgresConfigComponent := PostgresConfigComponent{}
	postgresConfig := postgresConfigComponent.Settings()
	assert.NotNil(t, postgresConfig)
	assert.NotEmpty(t, postgresConfig.Hostname)
}

func TestShouldFailToFindMigrationsPath(t *testing.T) {
	_, err := NewStorageMigrator("/thisdoesnotexist", nil)
	assert.NotNil(t, err)
}

func TestShouldFailToInitMigrationsSQL(t *testing.T) {
	// this is probably not cleanest for unit test, but / always exists and is a dir
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	mock.ExpectClose().WillReturnError(errors.New("unexpected error"))
	_, err = NewStorageMigrator("/", mockDB)
	assert.NotNil(t, err)
}

func TestShouldFailToMakeNewDB(t *testing.T) {
	postgresConfig := PostgresConfig{MigrationsPath: "/"}

	postgresConfigComponent := PostgresConfigComponent{}
	_, err := postgresConfigComponent.New(context.Background(), &postgresConfig)
	assert.NotNil(t, err)
}

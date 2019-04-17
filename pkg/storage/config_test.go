package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	postgresConfig := PostgresConfig{"localhost", "99", "me!", "mypassword!", "name"}
	assert.Equal(t, "Postgres", postgresConfig.Name())
}

func TestShouldReturnSame(t *testing.T) {
	postgresConfigComponent := PostgresConfigComponent{}
	postgresConfig := postgresConfigComponent.Settings()
	assert.NotNil(t, postgresConfig)
	assert.Empty(t, postgresConfig.DatabaseName)
}

func TestShouldMakeNewDB(t *testing.T) {
	called := false
	postgresConfig := PostgresConfig{}
	originalDBInitFn := dbInitFn
	defer func() {
		dbInitFn = originalDBInitFn
	}()

	dbInitFn = func(db *DB, ctx context.Context, c *PostgresConfig) error {
		called = true
		assert.Equal(t, postgresConfig, *c)
		return nil
	}

	postgresConfigComponent := PostgresConfigComponent{}
	_, _ = postgresConfigComponent.New(context.Background(), &postgresConfig)

	assert.True(t, called)
}

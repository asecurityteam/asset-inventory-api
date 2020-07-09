package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	postgresConfig := PostgresConfig{
		PartitionTTL: 365,
	}
	assert.Equal(t, "Postgres", postgresConfig.Name())
}

func TestNewPostgresComponent(t *testing.T) {
	postgres := NewPostgresComponent()
	assert.NotNil(t, postgres)
}

func TestShouldReturnSame(t *testing.T) {
	postgresConfigComponent := PostgresConfigComponent{}
	postgresConfig := postgresConfigComponent.Settings()
	assert.NotNil(t, postgresConfig)
	assert.NotEmpty(t, postgresConfig.URL)
}

func TestShouldFailToFindMigrationsPath(t *testing.T) {
	_, err := NewSchemaManager("/thisdoesnotexist", "")
	assert.NotNil(t, err)
}

func TestShouldFailToInitMigrationsSQL(t *testing.T) {
	// this is probably not cleanest for unit test, but / always exists and is a dir
	_, err := NewSchemaManager("/", "this is not a valid db url")
	assert.NotNil(t, err)
}

func TestShouldFailToConnectToInvalidDB(t *testing.T) {
	postgresConfig := PostgresConfig{MigrationsPath: "/", URL: "not a valid db url"}

	postgresConfigComponent := PostgresConfigComponent{}
	_, err := postgresConfigComponent.New(context.Background(), &postgresConfig, Primary)
	assert.NotNil(t, err)
}

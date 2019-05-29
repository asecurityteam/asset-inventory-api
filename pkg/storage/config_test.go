package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	postgresConfig := PostgresConfig{
		Hostname:     "localhost",
		Port:         "99",
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
	assert.Empty(t, postgresConfig.DatabaseName)
}

func TestShouldFailToMakeNewDB(t *testing.T) {
	postgresConfig := PostgresConfig{}

	postgresConfigComponent := PostgresConfigComponent{}
	_, err := postgresConfigComponent.New(context.Background(), &postgresConfig)
	assert.NotNil(t, err)
}

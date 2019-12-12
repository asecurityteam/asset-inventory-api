package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadName(t *testing.T) {
	postgresConfig := PostgresReadConfig{
		Hostname:     "localhost",
		Port:         "99",
		Username:     "me!",
		Password:     "mypassword!",
		DatabaseName: "name",
		PartitionTTL: 365,
	}
	assert.Equal(t, "PostgresRead", postgresConfig.Name())
}

func TestReadShouldReturnSame(t *testing.T) {
	postgresConfigComponent := PostgresConfigComponent{}
	postgresConfig := postgresConfigComponent.Settings()
	assert.NotNil(t, postgresConfig)
	assert.NotEmpty(t, postgresConfig.Hostname)
}

func TestReadShouldFailToMakeNewDB(t *testing.T) {
	postgresConfig := PostgresConfig{}

	postgresConfigComponent := PostgresConfigComponent{}
	_, err := postgresConfigComponent.New(context.Background(), &postgresConfig)
	assert.NotNil(t, err)
}

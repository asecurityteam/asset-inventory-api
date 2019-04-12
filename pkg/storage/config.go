package storage

import (
	"context"
)

// PostgresConfig contains the Postgres database configuration arguments
type PostgresConfig struct {
	Hostname     string
	Port         string
	Username     string
	Password     string
	DatabaseName string
}

// Name is used by the settings library to replace the default naming convention.
func (c *PostgresConfig) Name() string {
	return "Postgres"
}

// PostgresConfigComponent satisfies the settings library Component API,
// and may be used by the settings.NewComponent function.
type PostgresConfigComponent struct{}

// Settings populates a set of defaults if none are provided via config.
func (*PostgresConfigComponent) Settings() *PostgresConfig {
	return &PostgresConfig{}
}

// New constructs a PostgresConfig from a config.
func (*PostgresConfigComponent) New(_ context.Context, c *PostgresConfig) (*PostgresConfig, error) {
	// return &PostgresConfig{
	// 	Hostname: c.Hostname,
	// 	Port: c.Port,
	// 	Username: c.Username,
	// 	Password: c.Password,
	// 	DatabaseName: c.DatabaseName,
	// }, nil
	return c, nil
}

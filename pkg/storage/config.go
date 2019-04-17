package storage

import (
	"context"
)

// test seam
var dbInitFn = (*DB).Init

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

// New constructs a DB from a config.
func (*PostgresConfigComponent) New(ctx context.Context, c *PostgresConfig) (*DB, error) {
	db := &DB{}
	if err := dbInitFn(db, ctx, c); err != nil {
		return nil, err
	}
	return db, nil
}

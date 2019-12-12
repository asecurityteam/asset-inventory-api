package storage

import (
	"context"
)

// PostgresReadConfig contains the Postgres database configuration arguments for the ReadReplica
type PostgresReadConfig struct {
	Hostname     string
	Port         string
	Username     string
	Password     string
	DatabaseName string
	PartitionTTL int
}

// Name is used by the settings library to replace the default naming convention.
func (c *PostgresReadConfig) Name() string {
	return "PostgresRead"
}

// PostgresReadConfigComponent satisfies the settings library Component API,
// and may be used by the settings.NewComponent function.
type PostgresReadConfigComponent struct{}

// NewPostgresReadComponent generates a new PostgresReadConfigComponent
func NewPostgresReadComponent() *PostgresReadConfigComponent {
	return &PostgresReadConfigComponent{}
}

// Settings populates a set of defaults if none are provided via config.
func (*PostgresReadConfigComponent) Settings() *PostgresReadConfig {
	return &PostgresReadConfig{
		Hostname:     "localhost",
		Port:         "5432",
		Username:     "aiapi",
		DatabaseName: "aiapi",
		PartitionTTL: 360,
	}
}

// Unlike Master, Replica DB has no scripts and does not attempt to create the database as it can not by definition
func (*PostgresReadConfigComponent) New(ctx context.Context, c *PostgresReadConfig) (*DB, error) {
	db := &DB{}
	if err := db.Init(ctx, c.Hostname, c.Port, c.Username, c.Password, c.DatabaseName, c.PartitionTTL); err != nil {
		return nil, err
	}
	return db, nil
}

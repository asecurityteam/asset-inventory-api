package storage

import (
	"context"

	packr "github.com/gobuffalo/packr/v2"
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
	return &PostgresReadConfig{}
}

// New constructs a DB from a config.
func (*PostgresReadConfigComponent) New(ctx context.Context, c *PostgresReadConfig) (*DB, error) {
	scripts := packr.New("scripts", "../../scripts")
	db := &DB{
		scripts: scripts.FindString,
	}
	if err := db.Init(ctx, c.Hostname, c.Port, c.Username, c.Password, c.DatabaseName, c.PartitionTTL); err != nil {
		return nil, err
	}
	return db, nil
}

package storage

import (
	"context"

	packr "github.com/gobuffalo/packr/v2"
)

// PostgresConfig contains the Postgres database configuration arguments
type PostgresConfig struct {
	Hostname     string
	Port         string
	Username     string
	Password     string
	DatabaseName string
	PartitionTTL int
}

// Name is used by the settings library to replace the default naming convention.
func (c *PostgresConfig) Name() string {
	return "Postgres"
}

// PostgresConfigComponent satisfies the settings library Component API,
// and may be used by the settings.NewComponent function.
type PostgresConfigComponent struct{}

// NewPostgresComponent generates a PostgresConfigComponent
func NewPostgresComponent() *PostgresConfigComponent {
	return &PostgresConfigComponent{}
}

// Settings populates a set of defaults if none are provided via config.
func (*PostgresConfigComponent) Settings() *PostgresConfig {
	return &PostgresConfig{
		Hostname:     "localhost",
		Port:         "5432",
		Username:     "aiapi",
		DatabaseName: "aiapi",
		PartitionTTL: 360,
	}
}

// New constructs a DB from a config.
func (*PostgresConfigComponent) New(ctx context.Context, c *PostgresConfig) (*DB, error) {
	scripts := packr.New("scripts", "../../scripts")
	db := &DB{
		scripts: scripts.FindString,
	}
	if err := db.Init(ctx, c.Hostname, c.Port, c.Username, c.Password, c.DatabaseName, c.PartitionTTL); err != nil {
		return nil, err
	}
	return db, nil
}

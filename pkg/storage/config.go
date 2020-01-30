package storage

import (
	"context"
	"errors"
	"os"
)

// PostgresConfig contains the Postgres database configuration arguments
type PostgresConfig struct {
	Hostname         string
	Port             uint16
	Username         string
	Password         string
	DatabaseName     string
	PartitionTTL     int
	MinSchemaVersion uint
	MigrationsPath   string
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
		Hostname:         "localhost",
		Port:             5432,
		Username:         "aiapi",
		DatabaseName:     "aiapi",
		PartitionTTL:     360,
		MinSchemaVersion: 1,
		MigrationsPath:   "/db-migrations",
	}
}

// New constructs a DB from a config.
func (*PostgresConfigComponent) New(ctx context.Context, c *PostgresConfig) (*DB, error) {
	if mp, err := os.Stat(c.MigrationsPath); err != nil || !mp.IsDir() {
		return nil, errors.New("migrations path must exist and be a directory")
	}
	db := &DB{
		migrationsSourceURL: "file://" + c.MigrationsPath,
	}
	if err := db.Init(ctx, c.Hostname, c.Port, c.Username, c.Password, c.DatabaseName, c.PartitionTTL); err != nil {
		return nil, err
	}
	schemaVersion, err := db.GetSchemaVersion(ctx)
	if err != nil {
		return nil, err
	}
	if schemaVersion < c.MinSchemaVersion {
		if err := db.MigrateSchemaToVersion(ctx, c.MinSchemaVersion); err != nil {
			return nil, err
		}
	}

	return db, nil
}

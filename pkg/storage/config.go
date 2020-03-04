package storage

import (
	"context"
	"errors"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file" // used internally by migrate
	_ "github.com/lib/pq"                                // must remain here for sql lib to find the postgres driver
)

type connectionType int

// Database Connection types
const (
	Primary connectionType = iota
	Replica
)

// PostgresConfig contains the Postgres database configuration arguments
type PostgresConfig struct {
	URL              string
	ReplicaURL       string
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
		URL:              "postgres://aiapi:password@localhost/aiapi?sslmode=false",
		PartitionTTL:     360,
		MinSchemaVersion: MinimumSchemaVersion,
		MigrationsPath:   "/db-migrations",
	}
}

// NewSchemaManager generates a SchemaManager component based on settings
func NewSchemaManager(sourcePath string, datasourceURL string) (*SchemaManager, error) {
	if mp, err := os.Stat(sourcePath); err != nil || !mp.IsDir() {
		return nil, errors.New("migrator path must exist and be a directory")
	}
	sm := SchemaManager{
		DataSourceURL:    datasourceURL,
		MigrationsSource: "file://" + sourcePath,
	}
	migrator, err := migrate.New(sm.MigrationsSource, sm.DataSourceURL)
	if err != nil {
		return nil, err
	}
	sm.migrator = migrator
	return &sm, err
}

// New constructs a DB from a config.
func (*PostgresConfigComponent) New(ctx context.Context, c *PostgresConfig, t connectionType) (*DB, error) {
	db := &DB{}
	var err error
	url := c.URL
	if t == Replica {
		url = c.ReplicaURL
	}
	if err = db.Init(ctx, url, c.PartitionTTL); err != nil {
		return nil, err
	}
	return db, nil

}

package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // used internally by migrate
	_ "github.com/lib/pq"                                // must remain here for sql lib to find the postgres driver

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
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
		MinSchemaVersion: MinimumSchemaVersion,
		MigrationsPath:   "/db-migrations",
	}
}

// NewStorageMigrator constructs an instance of StorageMigrator implemented by psql driver + file migration back-end
func NewStorageMigrator(sourcePath string, db *sql.DB) (domain.StorageMigrator, error) {
	if mp, err := os.Stat(sourcePath); err != nil || !mp.IsDir() {
		return nil, errors.New("migrator path must exist and be a directory")
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, err
	}
	migrator, err := migrate.NewWithDatabaseInstance(
		"file://"+sourcePath,
		"postgres", driver)
	if err != nil {
		return nil, err
	}
	return migrator, nil
}

// New constructs a DB from a config.
func (*PostgresConfigComponent) New(ctx context.Context, c *PostgresConfig) (*DB, error) {
	db := &DB{}
	var err error
	if err = db.Init(ctx, c.Hostname, c.Port, c.Username, c.Password, c.DatabaseName, c.PartitionTTL); err != nil {
		return nil, err
	}
	db.migrator, err = NewStorageMigrator(c.MigrationsPath, db.sqldb)
	if err != nil {
		return nil, err
	}
	ver, err := db.GetSchemaVersion(context.Background())
	if err != nil {
		return nil, err
	}
	if ver >= c.MinSchemaVersion {
		return db, nil
	}
	// ErrNoChange means we are already on required version so we are good
	if err := db.MigrateSchemaToVersion(context.Background(), c.MinSchemaVersion); err != nil && err != migrate.ErrNoChange {
		return nil, err
	}
	return db, nil
}

package storage

import (
	"context"
	"errors"

	"github.com/golang-migrate/migrate/v4"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
)

type migrationDirection int

const (
	// used internally to designate migration direction
	up migrationDirection = iota
	down
)

const (
	// EmptySchemaVersion Version of database schema that cleans the database completely. Use cautiously!
	EmptySchemaVersion uint = 0
	// MinimumSchemaVersion Lowest version of database schema current code is able to handle
	MinimumSchemaVersion uint = 1
	// M1SchemaVersion Version of database schema with performance optimizations (M1) that allows back-fill to work
	M1SchemaVersion uint = 2
	// DualWritesSchemaVersion Lowest version of database schema that supports dual-writes (legacy and M1)
	DualWritesSchemaVersion uint = 3
	// ReadsFromNewSchemaVersion Lowest version of database schema that supports reads from M1 schema
	ReadsFromNewSchemaVersion uint = 4
	// NewSchemaOnlyVersion Lowest version that stops dual-writes in preparation to drop old schema
	NewSchemaOnlyVersion uint = 6
)

// SchemaManager is an abstraction layer for manipulating database schema backed by golang/migrate
type SchemaManager struct {
	DataSourceURL    string
	MigrationsSource string
	migrator         domain.StorageSchemaMigrator
}

// EnsureConnected checks if the migrator database connection is functioning properly and replaces it if needed
func (sm *SchemaManager) EnsureConnected() error {
	_, _, err := sm.migrator.Version()
	if err == nil || err == migrate.ErrNilVersion {
		return nil
	}
	// migrator has stale connection or is not initialized properly
	sm.migrator, err = migrate.New(sm.MigrationsSource, sm.DataSourceURL)
	return err
}

// ForceSchemaToVersion sets the database schema to specified version without running any migrations and clears dirty flag
func (sm *SchemaManager) ForceSchemaToVersion(ctx context.Context, version uint) error {
	// ensure we have an established connection before running any migrate commands
	err := sm.EnsureConnected()
	if err != nil {
		return err
	}
	return sm.migrator.Force(int(version))
}

// MigrateSchemaUp performs a database schema migration one version up
func (sm *SchemaManager) MigrateSchemaUp(ctx context.Context) (uint, error) {
	return sm.migrateSchema(ctx, up)
}

// MigrateSchemaDown performs a database schema rollback one version down
func (sm *SchemaManager) MigrateSchemaDown(ctx context.Context) (uint, error) {
	return sm.migrateSchema(ctx, down)
}

// MigrateSchemaToVersion performs one or more database migrations to bring schema to the specified version
func (sm *SchemaManager) MigrateSchemaToVersion(ctx context.Context, version uint) error {
	err := sm.EnsureConnected()
	if err != nil {
		return err
	}
	return sm.migrator.Migrate(version)
}

// GetSchemaVersion retrieves the current version of database schema
func (sm *SchemaManager) GetSchemaVersion(ctx context.Context) (uint, bool, error) {
	err := sm.EnsureConnected()
	if err != nil {
		return 0, false, err
	}
	v, dirty, err := sm.migrator.Version()
	if err == migrate.ErrNilVersion {
		// special handling for the version not being present
		return 0, dirty, nil
	}
	if err != nil {
		return 0, dirty, err
	}
	return v, dirty, nil
}

func (sm *SchemaManager) migrateSchema(ctx context.Context, d migrationDirection) (uint, error) {
	err := sm.EnsureConnected()
	if err != nil {
		return 0, err
	}
	switch d {
	case up:
		err = sm.migrator.Steps(1)
	case down:
		err = sm.migrator.Steps(-1)
	default:
		return 0, errors.New("unknown migration direction")
	}
	if err != nil {
		return 0, err
	}
	version, _, err := sm.GetSchemaVersion(ctx)
	if err != nil {
		return 0, err
	}
	return version, nil
}

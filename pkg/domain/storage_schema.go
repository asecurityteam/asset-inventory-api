package domain

import "context"

// StorageSchemaMigrator presents and abstraction over Database Schema migration
type StorageSchemaMigrator interface {
	Migrate(version uint) error
	Steps(steps int) error
	Version() (version uint, dirty bool, err error)
	Force(version int) error //NB, int version vs uint in Migrate
}

// SchemaVersionGetter is used to retrieve the current version of DB schema
type SchemaVersionGetter interface {
	GetSchemaVersion(ctx context.Context) (uint, bool, error)
}

// SchemaVersionForcer is used to force set database schema to specific version after failed migration
type SchemaVersionForcer interface {
	ForceSchemaToVersion(ctx context.Context, version uint) error
}

// SchemaMigratorToVersion is used to migrate database schema to specific version
type SchemaMigratorToVersion interface {
	MigrateSchemaToVersion(ctx context.Context, version uint) error
}

// SchemaMigratorUp is used to migrate database schema to newer version
type SchemaMigratorUp interface {
	MigrateSchemaUp(ctx context.Context) (uint, error)
}

// SchemaMigratorDown is used to migrate database schema to older version
type SchemaMigratorDown interface {
	MigrateSchemaDown(ctx context.Context) (uint, error)
}

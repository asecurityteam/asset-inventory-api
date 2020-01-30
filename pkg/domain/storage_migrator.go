package domain

// StorageMigrator presents and abstraction over Database Schema migration
type StorageMigrator interface {
	Migrate(version uint) error
	Steps(steps int) error
	Version() (version uint, dirty bool, err error)
}

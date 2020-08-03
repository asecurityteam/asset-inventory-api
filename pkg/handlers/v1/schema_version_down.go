package v1

import (
	"context"
	"fmt"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// SchemaVersionStepDownHandler handles requests for schema migration
type SchemaVersionStepDownHandler struct {
	LogFn    domain.LogFn
	Migrator domain.SchemaMigratorDown
}

// Handle handles the database schema change request
func (h *SchemaVersionStepDownHandler) Handle(ctx context.Context) (SchemaVersion, error) {
	newVersion, err := h.Migrator.MigrateSchemaDown(ctx)
	if err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
		return SchemaVersion{}, err
	}
	h.LogFn(ctx).Info(
		logs.MigrationSuccess{
			Reason: fmt.Sprintf("migrated down to schema version %d", newVersion)})
	return SchemaVersion{
		Version: newVersion,
	}, nil
}

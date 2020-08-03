package v1

import (
	"context"
	"fmt"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// SchemaVersionStepUpHandler handles requests for schema migration
type SchemaVersionStepUpHandler struct {
	LogFn    domain.LogFn
	Migrator domain.SchemaMigratorUp
}

// Handle handles the database schema change requests
func (h *SchemaVersionStepUpHandler) Handle(ctx context.Context) (SchemaVersion, error) {
	newVersion, err := h.Migrator.MigrateSchemaUp(ctx)
	if err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
		return SchemaVersion{}, err
	}
	h.LogFn(ctx).Info(
		logs.MigrationSuccess{
			Reason: fmt.Sprintf("migrated up to schema version %d", newVersion)})
	return SchemaVersion{
		Version: newVersion,
	}, nil
}

package v1

import (
	"context"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// SchemaVersionUpHandler handles requests for schema migration
type SchemaVersionUpHandler struct {
	LogFn    domain.LogFn
	Migrator domain.SchemaMigratorUp
}

// Handle handles the partition creation request
func (h *SchemaVersionUpHandler) Handle(ctx context.Context) (SchemaVersion, error) {
	newVersion, err := h.Migrator.MigrateSchemaUp(ctx)
	if err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
		return SchemaVersion{}, err
	}
	return SchemaVersion{
		Version: newVersion,
	}, nil
}

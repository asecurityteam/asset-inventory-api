package v1

import (
	"context"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// SchemaVersionDownHandler handles requests for schema migration
type SchemaVersionDownHandler struct {
	LogFn    domain.LogFn
	Migrator domain.SchemaMigratorDown
}

// Handle handles the partition creation request
func (h *SchemaVersionDownHandler) Handle(ctx context.Context) (SchemaVersion, error) {
	newVersion, err := h.Migrator.MigrateSchemaDown(ctx)
	if err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
		return SchemaVersion{}, err
	}
	return SchemaVersion{
		Version: newVersion,
	}, nil
}

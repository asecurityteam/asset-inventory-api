package v1

import (
	"context"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// ForceSchemaHandler defines a lambda handler for forcing database schema version after failed migration
type ForceSchemaHandler struct {
	LogFn               domain.LogFn
	SchemaVersionForcer domain.SchemaVersionForcer
}


// SchemaVersion represents a database schema version
type SchemaVersion struct {
	Version uint `json:"version"`
}

// Handle handles the call to force database schema version after failed migration
func (h *ForceSchemaHandler) Handle(ctx context.Context, input SchemaVersion) error {
	logger := h.LogFn(ctx)
	if e := h.SchemaVersionForcer.ForceSchemaToVersion(ctx, input.Version); e != nil {
		logger.Error(logs.StorageError{Reason: e.Error()})
		return e
	}
	return nil
}

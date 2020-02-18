package v1

import (
	"context"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// SchemaVersion represents an active database schema version
type SchemaVersion struct {
	Version uint `json:"version"`
}

// GetSchemaVersionHandler handles requests for getting the currently active database schema version
type GetSchemaVersionHandler struct {
	LogFn  domain.LogFn
	Getter domain.SchemaVersionGetter
}

// Handle handles the request for schema version
func (h *GetSchemaVersionHandler) Handle(ctx context.Context) (SchemaVersion, error) {
	currentVersion, err := h.Getter.GetSchemaVersion(ctx)
	if err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
		return SchemaVersion{}, err
	}
	return SchemaVersion{
		Version: currentVersion,
	}, nil
}

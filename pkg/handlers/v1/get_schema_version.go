package v1

import (
	"context"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)


// SchemaState represents current database schema version and state
type SchemaState struct {
	Version uint `json:"version"`
	Dirty bool `json:"dirty"`
}

// GetSchemaVersionHandler handles requests for getting the currently active database schema version
type GetSchemaVersionHandler struct {
	LogFn  domain.LogFn
	Getter domain.SchemaVersionGetter
}

// Handle handles the request for schema version
func (h *GetSchemaVersionHandler) Handle(ctx context.Context) (SchemaState, error) {
	currentVersion, dirty, err := h.Getter.GetSchemaVersion(ctx)
	if err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
		return SchemaState{}, err
	}
	return SchemaState{
		Version: currentVersion,
		Dirty: dirty,
	}, nil
}

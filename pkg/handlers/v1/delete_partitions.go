package v1

import (
	"context"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// DeletePartitionsInput contains the partition name to delete.
type DeletePartitionsInput struct {
	Name string `json:"name"`
}

// DeletePartitionsHandler handles requests for deleting partitions
type DeletePartitionsHandler struct {
	LogFn   domain.LogFn
	Deleter domain.PartitionsDeleter
}

// Handle handles the partition creation request
func (h *DeletePartitionsHandler) Handle(ctx context.Context, input DeletePartitionsInput) error {
	if err := h.Deleter.DeletePartitions(ctx, input.Name); err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
		return err
	}
	return nil
}

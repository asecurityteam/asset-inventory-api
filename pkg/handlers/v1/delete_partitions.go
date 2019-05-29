package v1

import (
	"context"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// DeletePartitionsInput has an optional value, days, which is the age of partitions to delete
type DeletePartitionsInput struct {
	Days int `json:"days"`
}

// DeletePartitionsOutput returns the number of partitions which were deleted
type DeletePartitionsOutput struct {
	Deleted int `json:"deleted"`
}

// DeletePartitionsHandler handles requests for deleting partitions
type DeletePartitionsHandler struct {
	LogFn   domain.LogFn
	Deleter domain.PartitionsDeleter
}

// Handle handles the partition creation request
func (h *DeletePartitionsHandler) Handle(ctx context.Context, input DeletePartitionsInput) (DeletePartitionsOutput, error) {
	result, err := h.Deleter.DeletePartitions(ctx, input.Days)
	if err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
		return DeletePartitionsOutput{}, err
	}
	return DeletePartitionsOutput{
		Deleted: result,
	}, nil
}

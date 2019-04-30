package v1

import (
	"context"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// CreatePartitionInput takes an optional timestamp for which to create the new partition
type CreatePartitionInput struct {
	Timestamp string
}

// CreatePartitionHandler handles requests for creating the next time-based partition
type CreatePartitionHandler struct {
	LogFn     domain.LogFn
	Generator domain.PartitionGenerator
}

// Handle handles the partition creation request
func (h *CreatePartitionHandler) Handle(ctx context.Context, input CreatePartitionInput) error {
	if input.Timestamp == "" {
		err := h.Generator.GeneratePartition(ctx)
		if err != nil {
			h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
		}
		return err
	}
	t, err := time.Parse(time.RFC3339, input.Timestamp)
	if err != nil {
		h.LogFn(ctx).Info(logs.InvalidInput{Reason: err.Error()})
		return InvalidInput{
			Cause: err,
			Field: "timestamp",
		}
	}
	err = h.Generator.GeneratePartitionWithTimestamp(ctx, t)
	if err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
	}
	return err
}

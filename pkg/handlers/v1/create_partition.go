package v1

import (
	"context"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// CreatePartitionInput has two optional values
// begin - the start date for the partition
// days - the duration in number of days for which the partition will capture data
type CreatePartitionInput struct {
	Begin time.Time `json:"begin"`
	Days  int       `json:"days"`
}

// CreatePartitionHandler handles requests for creating the next time-based partition
type CreatePartitionHandler struct {
	LogFn     domain.LogFn
	Generator domain.PartitionGenerator
}

// Handle handles the partition creation request
func (h *CreatePartitionHandler) Handle(ctx context.Context, input CreatePartitionInput) error {
	err := h.Generator.GeneratePartition(ctx, input.Begin, input.Days)
	if err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
	}
	return err
}

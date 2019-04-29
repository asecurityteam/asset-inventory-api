package v1

import (
	"context"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
)

// CreatePartitionHandler handles requests for creating the next time-based partition
type CreatePartitionHandler struct {
	LogFn     domain.LogFn
	Generator domain.PartitionGenerator
}

// Handle handles the partition creation request
func (h *CreatePartitionHandler) Handle(ctx context.Context) error {
	return h.Generator.GeneratePartition(ctx)
}

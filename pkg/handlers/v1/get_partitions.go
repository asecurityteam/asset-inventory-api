package v1

import (
	"context"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// GetPartitionsOutput returns the list of current partitions
type GetPartitionsOutput struct {
	Results []Partition `json:"results"`
}

// Partition represents a created database partition
type Partition struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Begin     time.Time `json:"begin"`
	End       time.Time `json:"end"`
	Count     int       `json:"count"`
}

// GetPartitionsHandler handles requests for getting the time based partitions
type GetPartitionsHandler struct {
	LogFn  domain.LogFn
	Getter domain.PartitionsGetter
}

// Handle handles the partition creation request
func (h *GetPartitionsHandler) Handle(ctx context.Context) (GetPartitionsOutput, error) {
	partitions, err := h.Getter.GetPartitions(ctx)
	if err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
		return GetPartitionsOutput{}, err
	}
	results := make([]Partition, 0, len(partitions))
	for _, partition := range partitions {
		results = append(results, Partition{
			Name:      partition.Name,
			CreatedAt: partition.CreatedAt,
			Begin:     partition.Begin,
			End:       partition.End,
			Count:     partition.Count,
		})
	}
	return GetPartitionsOutput{
		Results: results,
	}, nil
}

package v1

import (
	"context"
	"errors"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// BackFillEventsInput represents the incoming arguments to initiate schema back-fill
type BackFillEventsInput struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// BackFillEventsLocalHandler handles requests for local back-fill
type BackFillEventsLocalHandler struct {
	LogFn  domain.LogFn
	Runner domain.BackFillSchemaRunner
}

// Handle handles the partition creation request
func (h *BackFillEventsLocalHandler) Handle(ctx context.Context, input BackFillEventsInput) error {
	logger := h.LogFn(ctx)
	from, err := time.Parse(time.RFC3339Nano, input.From)
	if err != nil {
		logger.Info(logs.InvalidInput{Reason: err.Error()})
		return InvalidInput{Field: "from", Cause: err}
	}
	to, err := time.Parse(time.RFC3339Nano, input.To)
	if err != nil {
		logger.Info(logs.InvalidInput{Reason: err.Error()})
		return InvalidInput{Field: "to", Cause: err}
	}
	if from.After(to) {
		err = errors.New("invalid time range")
		logger.Info(logs.InvalidInput{Reason: err.Error()})
		return InvalidInput{Field: "from, to", Cause: err}
	}
	err = h.Runner.BackFillEventsLocally(ctx, from, to)
	if err != nil {
		h.LogFn(ctx).Error(logs.StorageError{Reason: err.Error()})
		return err
	}
	return nil
}

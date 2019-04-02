package v1

import (
	"context"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// CloudAssetChanges represents the incoming payload
type CloudAssetChanges struct {
	Changes      []NetworkChanges  `json:"changes"`
	ChangeTime   string            `json:"changeTime"`
	ResourceType string            `json:"resourceType"`
	AccountID    string            `json:"accountId"`
	Region       string            `json:"region"`
	ResourceID   string            `json:"resourceId"`
	Tags         map[string]string `json:"tags"`
}

// NetworkChanges detail the changes in ip addresses and host names for an asset
type NetworkChanges struct {
	PrivateIPAddresses []string `json:"privateIpAddresses"`
	PublicIPAddresses  []string `json:"publicIpAddresses"`
	Hostnames          []string `json:"hostnames"`
	ChangeType         string   `json:"changeType"`
}

// CloudInsertHandler defines a lambda handler for inserting new cloud asset or changes to existing cloud assets
type CloudInsertHandler struct {
	LogFn   domain.LogFn
	StatFn  domain.StatFn
	Storage domain.Storage
}

// Handle handles the insert operation for cloud assets
func (h *CloudInsertHandler) Handle(ctx context.Context, input CloudAssetChanges) error {
	logger := h.LogFn(ctx)

	changeTime, e := time.Parse(time.RFC3339Nano, input.ChangeTime)
	if e != nil {
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return InvalidInput{Field: "changeTime", Cause: e}
	}
	assetChanges := domain.CloudAssetChanges{
		ChangeTime:   changeTime,
		ResourceType: input.ResourceType,
		AccountID:    input.AccountID,
		Region:       input.Region,
		ResourceID:   input.ResourceID,
		Tags:         input.Tags,
		Changes:      make([]domain.NetworkChanges, 0, len(input.Changes)),
	}
	for _, val := range input.Changes {
		assetChanges.Changes = append(assetChanges.Changes, domain.NetworkChanges{
			PrivateIPAddresses: val.PrivateIPAddresses,
			PublicIPAddresses:  val.PublicIPAddresses,
			Hostnames:          val.Hostnames,
			ChangeType:         val.ChangeType,
		})
	}
	if e := h.Storage.StoreCloudAsset(ctx, assetChanges); e != nil {
		logger.Error(logs.StorageError{Reason: e.Error()})
		return e
	}
	return nil
}

package v1

import (
	"context"
	"fmt"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// CloudAssetFetchParameters represents the incoming payload for fetching a
// cloud asset
type CloudAssetFetchParameters struct {
	IPAddress string `json:"ipAddress"`
	Hostname  string `json:"hostname"`
	Timestamp string `json:"time"`
}

// CloudAssetDetails represent an asset and associated metadata
type CloudAssetDetails struct {
	PrivateIPAddresses []string          `json:"privateIpAddresses"`
	PublicIPAddresses  []string          `json:"publicIpAddresses"`
	Hostnames          []string          `json:"hostnames"`
	CreatedAt          string            `json:"createdAt"`
	DeletedAt          string            `json:"deletedAt"`
	ResourceType       string            `json:"resourceType"`
	AccountID          string            `json:"accountId"`
	Region             string            `json:"region"`
	ResourceID         string            `json:"resourceId"`
	Tags               map[string]string `json:"tags"`
}

// CloudFetchHandler defines a lambda handler for fetching a cloud asset
type CloudFetchHandler struct {
	LogFn   domain.LogFn
	StatFn  domain.StatFn
	Storage domain.Storage
}

// Handle handles the fetch operation for cloud assets
func (h *CloudFetchHandler) Handle(ctx context.Context, input CloudAssetFetchParameters) (CloudAssetDetails, error) {
	logger := h.LogFn(ctx)

	ts, e := time.Parse(time.RFC3339Nano, input.Timestamp)
	if e != nil {
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return CloudAssetDetails{}, InvalidInput{Field: "time", Cause: e}
	}

	if input.Hostname == "" && input.IPAddress == "" {
		e = fmt.Errorf("hostname and ipAddress cannot both be empty")
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return CloudAssetDetails{}, InvalidInput{Field: "hostname,ipAddress", Cause: e}
	}

	asset, e := h.Storage.FetchCloudAsset(ctx, input.Hostname, input.IPAddress, ts)
	if e != nil {
		logger.Error(logs.StorageError{Reason: e.Error()})
		return CloudAssetDetails{}, e
	}

	output := CloudAssetDetails{
		PrivateIPAddresses: asset.PrivateIPAddresses,
		PublicIPAddresses:  asset.PublicIPAddresses,
		Hostnames:          asset.Hostnames,
		CreatedAt:          asset.CreatedAt.Format(time.RFC3339Nano),
		DeletedAt:          asset.DeletedAt.Format(time.RFC3339Nano),
		ResourceType:       asset.ResourceType,
		AccountID:          asset.AccountID,
		Region:             asset.Region,
		ResourceID:         asset.ResourceID,
		Tags:               asset.Tags,
	}

	return output, nil
}

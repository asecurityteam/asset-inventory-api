package v1

import (
	"context"
	"fmt"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// CloudAssets represents a list of assets
type CloudAssets struct {
	Assets []CloudAssetDetails `json:"assets"`
}

// CloudAssetDetails represent an asset and associated attributes
type CloudAssetDetails struct {
	PrivateIPAddresses []string          `json:"privateIpAddresses"`
	PublicIPAddresses  []string          `json:"publicIpAddresses"`
	Hostnames          []string          `json:"hostnames"`
	ResourceType       string            `json:"resourceType"`
	AccountID          string            `json:"accountId"`
	Region             string            `json:"region"`
	ARN                string            `json:"arn"`
	Tags               map[string]string `json:"tags"`
}

// CloudAssetFetchByIPParameters represents the incoming payload for fetching cloud assets by IP address
type CloudAssetFetchByIPParameters struct {
	IPAddress string `json:"ipAddress"`
	Timestamp string `json:"time"`
}

// CloudAssetFetchByHostnameParameters represents the incoming payload for fetching cloud assets by hostname
type CloudAssetFetchByHostnameParameters struct {
	Hostname  string `json:"hostname"`
	Timestamp string `json:"time"`
}

// CloudFetchByIPHandler defines a lambda handler for fetching cloud assets with a given IP address
type CloudFetchByIPHandler struct {
	LogFn   domain.LogFn
	StatFn  domain.StatFn
	Fetcher domain.CloudAssetByIPFetcher
}

// Handle handles fetching cloud assets by IP address
func (h *CloudFetchByIPHandler) Handle(ctx context.Context, input CloudAssetFetchByIPParameters) (CloudAssets, error) {
	logger := h.LogFn(ctx)

	ts, e := time.Parse(time.RFC3339Nano, input.Timestamp)
	if e != nil {
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return CloudAssets{}, InvalidInput{Field: "time", Cause: e}
	}

	if input.IPAddress == "" {
		e = fmt.Errorf("ipAddress cannot be empty")
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return CloudAssets{}, InvalidInput{Field: "ipAddress", Cause: e}
	}

	assets, e := h.Fetcher.FetchByIP(ctx, ts, input.IPAddress)
	if e != nil {
		logger.Error(logs.StorageError{Reason: e.Error()})
		return CloudAssets{}, e
	}
	if len(assets) == 0 {
		return CloudAssets{}, NotFound{ID: input.IPAddress}
	}

	return extractOutput(assets), nil
}

// CloudFetchByHostnameHandler defines a lambda handler for fetching cloud assets with a given hostname
type CloudFetchByHostnameHandler struct {
	LogFn   domain.LogFn
	StatFn  domain.StatFn
	Fetcher domain.CloudAssetByHostnameFetcher
}

// Handle handles fetching cloud assets by hostname
func (h *CloudFetchByHostnameHandler) Handle(ctx context.Context, input CloudAssetFetchByHostnameParameters) (CloudAssets, error) {
	logger := h.LogFn(ctx)

	ts, e := time.Parse(time.RFC3339Nano, input.Timestamp)
	if e != nil {
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return CloudAssets{}, InvalidInput{Field: "time", Cause: e}
	}

	if input.Hostname == "" {
		e = fmt.Errorf("hostname cannot be empty")
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return CloudAssets{}, InvalidInput{Field: "hostname", Cause: e}
	}

	assets, e := h.Fetcher.FetchByHostname(ctx, ts, input.Hostname)
	if e != nil {
		logger.Error(logs.StorageError{Reason: e.Error()})
		return CloudAssets{}, e
	}
	if len(assets) == 0 {
		return CloudAssets{}, NotFound{ID: input.Hostname}
	}

	return extractOutput(assets), nil
}

func extractOutput(assets []domain.CloudAssetDetails) CloudAssets {
	cloudAssets := CloudAssets{
		Assets: make([]CloudAssetDetails, len(assets)),
	}
	for i, asset := range assets {
		hostnames := asset.Hostnames
		if len(hostnames) == 0 {
			hostnames = make([]string, 0)
		}
		privateIPAddresses := asset.PrivateIPAddresses
		if len(privateIPAddresses) == 0 {
			privateIPAddresses = make([]string, 0)
		}
		publicIPAddresses := asset.PublicIPAddresses
		if len(publicIPAddresses) == 0 {
			publicIPAddresses = make([]string, 0)
		}
		tags := asset.Tags
		if len(tags) == 0 {
			tags = make(map[string]string)
		}

		cloudAssets.Assets[i] = CloudAssetDetails{
			PrivateIPAddresses: privateIPAddresses,
			PublicIPAddresses:  publicIPAddresses,
			Hostnames:          hostnames,
			ResourceType:       asset.ResourceType,
			AccountID:          asset.AccountID,
			Region:             asset.Region,
			ARN:                asset.ARN,
			Tags:               tags,
		}
	}
	return cloudAssets
}

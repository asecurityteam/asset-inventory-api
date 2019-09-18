package v1

import (
	"context"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

const (
	awsEC2 = "AWS::EC2::Instance"
	awsELB = "AWS::ElasticLoadBalancing::LoadBalancer"
	awsALB = "AWS::ElasticLoadBalancingV2::LoadBalancer"
)

// CloudAssets represents a list of assets
type CloudAssets struct {
	Assets []CloudAssetDetails `json:"assets"`
}

// PagedCloudAssets represents a list of assets with next page token for bulk requests
type PagedCloudAssets struct {
	CloudAssets
	NextPageToken string `json:"nextPageToken"`
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

// CloudAssetFetchAllByTimestampParameters represents the incoming payload for bulk fetching cloud assets for point in time with optional pagination
type CloudAssetFetchAllByTimestampParameters struct {
	Timestamp string `json:"time"`
	Count     uint   `json:"count"`
	Offset    uint   `json:"offset"`
	Type      string `json:"type"`
}

func (p *CloudAssetFetchAllByTimestampParameters) toNextPageToken() (string, error) {
	nextPageParameters := CloudAssetFetchAllByTimestampParameters{
		Timestamp: p.Timestamp,
		Count:     p.Count,
		Offset:    p.Offset + p.Count,
		Type:      p.Type,
	}
	js, err := json.Marshal(nextPageParameters)
	if err != nil {
		return "", err
	}
	token := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(js)
	return token, nil
}

func fetchAllByTimeStampParametersForToken(token string) (*CloudAssetFetchAllByTimestampParameters, error) {
	js, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(token)
	if err != nil {
		return nil, err
	}
	ret := CloudAssetFetchAllByTimestampParameters{}
	err = json.Unmarshal(js, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

// CloudAssetFetchAllByTimeStampPageParameters represents the request for subsequent pages of bulk fetching cloud assets
type CloudAssetFetchAllByTimeStampPageParameters struct {
	PageToken string `json:"pageToken"`
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

func validateAssetType(input string) (string, error) {
	switch input {
	case awsEC2, awsELB, awsALB:
		return input, nil
	default:
		return "", fmt.Errorf("unknown asset type %s", input)
	}
}

// CloudFetchAllAssetsByTimeHandler defines a lambda handler for bulk fetching cloud assets known at specific point in time
type CloudFetchAllAssetsByTimeHandler struct {
	LogFn   domain.LogFn
	StatFn  domain.StatFn
	Fetcher domain.CloudAllAssetsByTimeFetcher
}

// Handle handles fetching cloud assets with pagination
func (h *CloudFetchAllAssetsByTimeHandler) Handle(ctx context.Context, input CloudAssetFetchAllByTimestampParameters) (PagedCloudAssets, error) {
	logger := h.LogFn(ctx)

	ts, e := time.Parse(time.RFC3339Nano, input.Timestamp)
	if e != nil {
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return PagedCloudAssets{}, InvalidInput{Field: "time", Cause: e}
	}

	if input.Count == 0 {
		e = errors.New("missing or malformed required parameter count")
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return PagedCloudAssets{}, InvalidInput{Field: "count", Cause: e}
	}

	assetType, e := validateAssetType(input.Type)
	if e != nil {
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return PagedCloudAssets{}, InvalidInput{Field: "type", Cause: e}
	}

	var offset uint = 0 // do not offset as this is the first page
	assets, e := h.Fetcher.FetchAll(ctx, ts, input.Count, offset, assetType)
	if e != nil {
		logger.Error(logs.StorageError{Reason: e.Error()})
		return PagedCloudAssets{}, e
	}
	if len(assets) == 0 {
		return PagedCloudAssets{}, NotFound{ID: "any"}
	}

	nextPageToken, e := input.toNextPageToken()
	if e != nil {
		logger.Error(logs.StorageError{Reason: e.Error()})
	}
	return PagedCloudAssets{extractOutput(assets), nextPageToken}, nil
}

// CloudFetchAllAssetsByTimePageHandler defines a lambda handler for bulk fetching subsequent pages of cloud assets known at specific point in time
type CloudFetchAllAssetsByTimePageHandler struct {
	LogFn   domain.LogFn
	StatFn  domain.StatFn
	Fetcher domain.CloudAllAssetsByTimeFetcher
}

// Handle handles subsequent page fetching of cloud assets
func (h *CloudFetchAllAssetsByTimePageHandler) Handle(ctx context.Context, input CloudAssetFetchAllByTimeStampPageParameters) (PagedCloudAssets, error) {
	logger := h.LogFn(ctx)
	params, e := fetchAllByTimeStampParametersForToken(input.PageToken)
	if e != nil {
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return PagedCloudAssets{}, InvalidInput{Field: "PageToken", Cause: e}
	}
	ts, e := time.Parse(time.RFC3339Nano, params.Timestamp)
	if e != nil {
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return PagedCloudAssets{}, InvalidInput{Field: "PageToken", Cause: e}
	}

	if params.Count == 0 {
		e = errors.New("missing or malformed required parameter count")
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return PagedCloudAssets{}, InvalidInput{Field: "count", Cause: e}
	}

	if params.Offset == 0 {
		e = errors.New("missing or malformed required parameter offset")
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return PagedCloudAssets{}, InvalidInput{Field: "offset", Cause: e}
	}

	assetType, e := validateAssetType(params.Type)
	if e != nil {
		logger.Info(logs.InvalidInput{Reason: e.Error()})
		return PagedCloudAssets{}, InvalidInput{Field: "type", Cause: e}
	}

	logger.Info(fmt.Sprintf("count: %d offset: %d, type: %s ,ts %v", params.Count, params.Offset, assetType, ts))
	assets, e := h.Fetcher.FetchAll(ctx, ts, params.Count, params.Offset, assetType)
	if e != nil {
		logger.Error(logs.StorageError{Reason: e.Error()})
		return PagedCloudAssets{}, e
	}
	logger.Info(fmt.Sprintf("count: %d offset: %d, type: %s ,ts %v, r %d", params.Count, params.Offset, assetType, ts, len(assets)))
	if len(assets) == 0 {
		return PagedCloudAssets{}, NotFound{ID: "any"}
	}

	nextPageToken, e := params.toNextPageToken()
	return PagedCloudAssets{extractOutput(assets), nextPageToken}, nil
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

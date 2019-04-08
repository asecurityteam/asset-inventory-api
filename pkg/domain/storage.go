package domain

import (
	"context"
	"time"
)

// Storage interface provides functions for inserting and retrieving assets
type Storage interface {
	StoreCloudAsset(context.Context, CloudAssetChanges) error
	FetchCloudAsset(ctx context.Context, hostname string, ipAddress string, timestamp time.Time) (CloudAssetDetails, error)
}

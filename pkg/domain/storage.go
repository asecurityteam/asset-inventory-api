package domain

import (
	"context"
)

// Storage interface provides functions for inserting and retrieving assets
type Storage interface {
	StoreCloudAsset(context.Context, CloudAssetChanges) error
}

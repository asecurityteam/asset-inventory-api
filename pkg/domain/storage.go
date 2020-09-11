package domain

import (
	"context"
	"time"
)

// CloudAssetStorer interface provides functions for inserting cloud assets
type CloudAssetStorer interface {
	Store(context.Context, CloudAssetChanges) error
}

// CloudAssetByIPFetcher fetches details for a cloud asset with a given IP address at a point in time
type CloudAssetByIPFetcher interface {
	FetchByIP(ctx context.Context, when time.Time, ipAddress string) ([]CloudAssetDetails, error)
}

// CloudAssetByHostnameFetcher fetches details for a cloud asset with a given hostname at a point in time
type CloudAssetByHostnameFetcher interface {
	FetchByHostname(ctx context.Context, when time.Time, hostname string) ([]CloudAssetDetails, error)
}

// CloudAssetByResourceIDFetcher fetches details for a cloud asset with a given resource ID at a point in time
type CloudAssetByResourceIDFetcher interface {
	FetchByResourceID(ctx context.Context, when time.Time, resid string) ([]CloudAssetDetails, error)
}

// CloudAllAssetsByTimeFetcher fetches details for all cloud assets based on limit and optional offset with a given point in time
type CloudAllAssetsByTimeFetcher interface {
	FetchAll(ctx context.Context, when time.Time, count uint, offset uint, assetType string) ([]CloudAssetDetails, error)
}

// EventExportHandler handles exporting a single event during export
type EventExportHandler interface {
	Handle(changes CloudAssetChanges) error
}

// AccountOwnerStorer interface provides functions for updating account owner and champions
type AccountOwnerStorer interface {
	StoreAccountOwner(context.Context, AccountOwner) error
}

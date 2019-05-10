package domain

import (
	"context"
	"fmt"
	"time"
)

// PartitionGenerator is used to generate the next time-based partition
type PartitionGenerator interface {
	GeneratePartition(context.Context) error
	GeneratePartitionWithTimestamp(context.Context, time.Time) error
}

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

// PartitionConflict is used to indicate a partition exists which overlaps with a partition requested to be created
type PartitionConflict struct {
	Name string
}

func (e PartitionConflict) Error() string {
	return fmt.Sprintf("A partition already exists which overlaps witht the requested partition, %s", e.Name)
}

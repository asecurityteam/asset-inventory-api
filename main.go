package main

import (
	"context"
	"os"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	v1 "github.com/asecurityteam/asset-inventory-api/pkg/handlers/v1"
	"github.com/asecurityteam/asset-inventory-api/pkg/storage"
	"github.com/asecurityteam/serverfull"
	"github.com/asecurityteam/settings"
)

func main() {
	ctx := context.Background()
	source, err := settings.NewEnvSource(os.Environ())
	if err != nil {
		panic(err.Error())
	}

	postgresConfigComponent := &storage.PostgresConfigComponent{}
	dbStorage := new(storage.DB)
	if err = settings.NewComponent(ctx, source, postgresConfigComponent, dbStorage); err != nil {
		panic(err.Error())
	}
	insert := &v1.CloudInsertHandler{
		LogFn:            domain.LoggerFromContext,
		StatFn:           domain.StatFromContext,
		CloudAssetStorer: dbStorage,
	}
	fetchByIP := &v1.CloudFetchByIPHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: dbStorage,
	}
	fetchByHostname := &v1.CloudFetchByHostnameHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: dbStorage,
	}
	fetchAllAssetsByTime := &v1.CloudFetchAllAssetsByTimeHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: dbStorage,
	}
	fetchAllAssetsByTimePage := &v1.CloudFetchAllAssetsByTimePageHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: dbStorage,
	}
	createPartition := &v1.CreatePartitionHandler{
		LogFn:     domain.LoggerFromContext,
		Generator: dbStorage,
	}
	getPartitions := &v1.GetPartitionsHandler{
		LogFn:  domain.LoggerFromContext,
		Getter: dbStorage,
	}
	deletePartitions := &v1.DeletePartitionsHandler{
		LogFn:   domain.LoggerFromContext,
		Deleter: dbStorage,
	}
	handlers := map[string]serverfull.Function{
		"insert":                     serverfull.NewFunction(insert.Handle),
		"fetchByIP":                  serverfull.NewFunction(fetchByIP.Handle),
		"fetchByHostname":            serverfull.NewFunction(fetchByHostname.Handle),
		"fetchAllAssetsByTime":       serverfull.NewFunction(fetchAllAssetsByTime.Handle),
		"fetchMoreAssetsByPageToken": serverfull.NewFunction(fetchAllAssetsByTimePage.Handle),
		"createPartition":            serverfull.NewFunction(createPartition.Handle),
		"getPartitions":              serverfull.NewFunction(getPartitions.Handle),
		"deletePartitions":           serverfull.NewFunction(deletePartitions.Handle),
	}

	fetcher := &serverfull.StaticFetcher{Functions: handlers}
	if err := serverfull.Start(ctx, source, fetcher); err != nil {
		panic(err.Error())
	}
}

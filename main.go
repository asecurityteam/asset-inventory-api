package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	v1 "github.com/asecurityteam/asset-inventory-api/pkg/handlers/v1"
	"github.com/asecurityteam/asset-inventory-api/pkg/storage"
	"github.com/asecurityteam/serverfull"
	"github.com/asecurityteam/settings"
)

// testing

type config struct {
	PostgresConfig *storage.PostgresConfig
}

func (*config) Name() string {
	return "AIAPI"
}

type component struct {
	PostgresConfig *storage.PostgresConfigComponent
}

func newComponent() *component {
	return &component{
		PostgresConfig: storage.NewPostgresComponent(),
	}
}

func (c *component) Settings() *config {
	return &config{
		PostgresConfig: c.PostgresConfig.Settings(),
	}
}

func (c *component) New(ctx context.Context, conf *config) (func(context.Context, settings.Source) error, error) {
	primaryStorage, err := c.PostgresConfig.New(ctx, conf.PostgresConfig, storage.Primary)
	if err != nil {
		return nil, err
	}
	replicaStorage, err := c.PostgresConfig.New(ctx, conf.PostgresConfig, storage.Replica)
	if err != nil || replicaStorage == nil { //if the replica is not properly configured - fall back to primary
		replicaStorage = primaryStorage
	}

	schemaManager, err := storage.NewSchemaManager(conf.PostgresConfig.MigrationsPath, conf.PostgresConfig.URL)
	if err != nil {
		return nil, err
	}
	ver, err := schemaManager.GetSchemaVersion(context.Background())
	if err != nil {
		return nil, err
	}
	if ver < conf.PostgresConfig.MinSchemaVersion {
		// ErrNoChange means we are already on required version so we are good
		err := schemaManager.MigrateSchemaToVersion(context.Background(), conf.PostgresConfig.MinSchemaVersion)
		if err != nil && err != migrate.ErrNoChange {
			return nil, err
		}
	}

	insert := &v1.CloudInsertHandler{
		LogFn:            domain.LoggerFromContext,
		StatFn:           domain.StatFromContext,
		CloudAssetStorer: primaryStorage,
	}
	fetchByIP := &v1.CloudFetchByIPHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: replicaStorage,
	}
	fetchByHostname := &v1.CloudFetchByHostnameHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: replicaStorage,
	}
	fetchByArnID := &v1.CloudFetchByARNIDHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: replicaStorage,
	}
	fetchAllAssetsByTime := &v1.CloudFetchAllAssetsByTimeHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: replicaStorage,
	}
	fetchAllAssetsByTimePage := &v1.CloudFetchAllAssetsByTimePageHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: replicaStorage,
	}
	createPartition := &v1.CreatePartitionHandler{
		LogFn:     domain.LoggerFromContext,
		Generator: primaryStorage,
	}
	getPartitions := &v1.GetPartitionsHandler{
		LogFn:  domain.LoggerFromContext,
		Getter: primaryStorage,
	}
	deletePartitions := &v1.DeletePartitionsHandler{
		LogFn:   domain.LoggerFromContext,
		Deleter: primaryStorage,
	}
	getSchemaVersion := &v1.GetSchemaVersionHandler{
		LogFn:  domain.LoggerFromContext,
		Getter: schemaManager,
	}
	schemaVersionStepUp := &v1.SchemaVersionStepUpHandler{
		LogFn:    domain.LoggerFromContext,
		Migrator: schemaManager,
	}
	schemaVersionStepDown := &v1.SchemaVersionStepDownHandler{
		LogFn:    domain.LoggerFromContext,
		Migrator: schemaManager,
	}
	backFillLocally := &v1.BackFillEventsLocalHandler{
		LogFn:  domain.LoggerFromContext,
		Runner: primaryStorage,
	}
	forceSchemaVersion := &v1.ForceSchemaHandler{
		LogFn:               domain.LoggerFromContext,
		SchemaVersionForcer: schemaManager,
	}
	insertAccountOwner := &v1.AccountOwnerInsertHandler{
		LogFn:              domain.LoggerFromContext,
		StatFn:             domain.StatFromContext,
		AccountOwnerStorer: primaryStorage,
	}

	handlers := map[string]serverfull.Function{
		"insert":                     serverfull.NewFunction(insert.Handle),
		"fetchByIP":                  serverfull.NewFunction(fetchByIP.Handle),
		"fetchByHostname":            serverfull.NewFunction(fetchByHostname.Handle),
		"fetchByArnID":               serverfull.NewFunction(fetchByArnID.Handle),
		"fetchAllAssetsByTime":       serverfull.NewFunction(fetchAllAssetsByTime.Handle),
		"fetchMoreAssetsByPageToken": serverfull.NewFunction(fetchAllAssetsByTimePage.Handle),
		"createPartition":            serverfull.NewFunction(createPartition.Handle),
		"getPartitions":              serverfull.NewFunction(getPartitions.Handle),
		"deletePartitions":           serverfull.NewFunction(deletePartitions.Handle),
		"getSchemaVersion":           serverfull.NewFunction(getSchemaVersion.Handle),
		"schemaVersionStepUp":        serverfull.NewFunction(schemaVersionStepUp.Handle),
		"schemaVersionStepDown":      serverfull.NewFunction(schemaVersionStepDown.Handle),
		"backFillLocally":            serverfull.NewFunction(backFillLocally.Handle),
		"forceSchemaVersion":         serverfull.NewFunction(forceSchemaVersion.Handle),
		"insertAccountOwner":         serverfull.NewFunction(insertAccountOwner.Handle),
	}

	fetcher := &serverfull.StaticFetcher{Functions: handlers}
	return func(ctx context.Context, source settings.Source) error {
		return serverfull.Start(ctx, source, fetcher)
	}, nil
}

func main() {
	ctx := context.Background()
	source, err := settings.NewEnvSource(os.Environ())
	if err != nil {
		panic(err.Error())
	}
	runner := new(func(context.Context, settings.Source) error)
	cmp := newComponent()
	fs := flag.NewFlagSet("asset-inventory-api", flag.ContinueOnError)
	fs.Usage = func() {}
	if err = fs.Parse(os.Args[1:]); err == flag.ErrHelp {
		sg, _ := settings.GroupFromComponent(cmp)
		fmt.Println("Usage:")
		fmt.Println(settings.ExampleEnvGroups([]settings.Group{sg}))
		return
	}
	if err = settings.NewComponent(ctx, source, cmp, runner); err != nil {
		panic(err.Error())
	}
	if err := (*runner)(ctx, source); err != nil {
		panic(err.Error())
	}
}

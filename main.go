package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	v1 "github.com/asecurityteam/asset-inventory-api/pkg/handlers/v1"
	"github.com/asecurityteam/asset-inventory-api/pkg/storage"
	"github.com/asecurityteam/serverfull"
	"github.com/asecurityteam/settings"
)

type config struct {
	PostgresConfig     *storage.PostgresConfig
	PostgresReadConfig *storage.PostgresReadConfig
}

func (*config) Name() string {
	return "AIAPI"
}

type component struct {
	PostgresConfig     *storage.PostgresConfigComponent
	PostgresReadConfig *storage.PostgresReadConfigComponent
}

func newComponent() *component {
	return &component{
		PostgresConfig:     storage.NewPostgresComponent(),
		PostgresReadConfig: storage.NewPostgresReadComponent(),
	}
}

func (c *component) Settings() *config {
	return &config{
		PostgresConfig:     c.PostgresConfig.Settings(),
		PostgresReadConfig: c.PostgresReadConfig.Settings(),
	}
}

func (c *component) New(ctx context.Context, conf *config) (func(context.Context, settings.Source) error, error) {
	dbStorage, err := c.PostgresConfig.New(ctx, conf.PostgresConfig)
	if err != nil {
		return nil, err
	}
	readDbStorage, err := c.PostgresReadConfig.New(ctx, conf.PostgresReadConfig)
	if err != nil {
		return nil, err
	}

	insert := &v1.CloudInsertHandler{
		LogFn:            domain.LoggerFromContext,
		StatFn:           domain.StatFromContext,
		CloudAssetStorer: dbStorage,
	}
	fetchByIP := &v1.CloudFetchByIPHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: readDbStorage,
	}
	fetchByHostname := &v1.CloudFetchByHostnameHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: readDbStorage,
	}
	fetchAllAssetsByTime := &v1.CloudFetchAllAssetsByTimeHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: readDbStorage,
	}
	fetchAllAssetsByTimePage := &v1.CloudFetchAllAssetsByTimePageHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: readDbStorage,
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
	getSchemaVersion := &v1.GetSchemaVersionHandler{
		LogFn:  domain.LoggerFromContext,
		Getter: dbStorage,
	}
	schemaVersionStepUp := &v1.SchemaVersionStepUpHandler{
		LogFn:    domain.LoggerFromContext,
		Migrator: dbStorage,
	}
	schemaVersionStepDown := &v1.SchemaVersionStepDownHandler{
		LogFn:    domain.LoggerFromContext,
		Migrator: dbStorage,
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
		"getSchemaVersion":           serverfull.NewFunction(getSchemaVersion.Handle),
		"schemaVersionStepUp":        serverfull.NewFunction(schemaVersionStepUp.Handle),
		"schemaVersionStepDown":      serverfull.NewFunction(schemaVersionStepDown.Handle),
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

package main

import (
	"context"
	"os"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	v1 "github.com/asecurityteam/asset-inventory-api/pkg/handlers/v1"
	"github.com/asecurityteam/asset-inventory-api/pkg/storage"
	serverfull "github.com/asecurityteam/serverfull/pkg"
	serverfulldomain "github.com/asecurityteam/serverfull/pkg/domain"
	"github.com/asecurityteam/settings"
	"github.com/aws/aws-lambda-go/lambda"
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
		LogFn:  domain.LoggerFromContext,
		StatFn: domain.StatFromContext,
		CloudAssetStorer: &storage.NopStorer{
			LogFn: domain.LoggerFromContext,
		},
	}
	fetchByIP := &v1.CloudFetchByIPHandler{
		LogFn:  domain.LoggerFromContext,
		StatFn: domain.StatFromContext,
		Fetcher: &storage.NopFetcher{
			LogFn: domain.LoggerFromContext,
		},
	}
	fetchByHostname := &v1.CloudFetchByHostnameHandler{
		LogFn:  domain.LoggerFromContext,
		StatFn: domain.StatFromContext,
		Fetcher: &storage.NopFetcher{
			LogFn: domain.LoggerFromContext,
		},
	}
	handlers := map[string]serverfulldomain.Handler{
		"insert":          lambda.NewHandler(insert.Handle),
		"fetchByIP":       lambda.NewHandler(fetchByIP.Handle),
		"fetchByHostname": lambda.NewHandler(fetchByHostname.Handle),
	}

	rt, err := serverfull.NewStatic(ctx, source, handlers)
	if err != nil {
		panic(err.Error())
	}
	if err := rt.Run(); err != nil {
		panic(err.Error())
	}
}

package main

import (
	"context"
	"os"

	v1 "github.com/asecurityteam/asset-inventory-api/pkg/handlers/v1"
	"github.com/asecurityteam/asset-inventory-api/pkg/storage"
	"github.com/asecurityteam/runhttp"
	serverfull "github.com/asecurityteam/serverfull/pkg"
	serverfulldomain "github.com/asecurityteam/serverfull/pkg/domain"
	"github.com/asecurityteam/settings"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	ctx := context.Background()

	insert := &v1.CloudInsertHandler{
		LogFn:   runhttp.LoggerFromContext,
		StatFn:  runhttp.StatFromContext,
		Storage: &db,
	}
	handlers := map[string]serverfulldomain.Handler{
		"insert": lambda.NewHandler(insert.Handle),
	}
	source, err := settings.NewEnvSource(os.Environ())
	if err != nil {
		panic(err.Error())
	}

	postgresConfigComponent := &storage.PostgresConfigComponent{}
	postgresSettings := new(storage.PostgresSettings)
	err = settings.NewComponent(ctx, source, postgresConfigComponent, postgresSettings)
	if err != nil {
		panic(err.Error())
	}
	db := storage.DB{}
	if err := db.Init(ctx); err != nil {
		panic(err.Error())
	}

	rt, err := serverfull.NewStatic(ctx, source, handlers)
	if err != nil {
		panic(err.Error())
	}
	if err := rt.Run(); err != nil {
		panic(err.Error())
	}
}

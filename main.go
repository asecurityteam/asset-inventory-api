package main

import (
	"context"
	"os"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	v1 "github.com/asecurityteam/asset-inventory-api/pkg/handlers/v1"
	"github.com/asecurityteam/runhttp"
	serverfull "github.com/asecurityteam/serverfull/pkg"
	serverfulldomain "github.com/asecurityteam/serverfull/pkg/domain"
	"github.com/asecurityteam/settings"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	ctx := context.Background()

	db := v1.DB{}
	err := db.Init(ctx); err != nil {
		panic(err.Error())
	}
	if err != nil {
		panic(err.Error())
	}

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
	rt, err := serverfull.NewStatic(ctx, source, handlers)
	if err != nil {
		panic(err.Error())
	}
	if err := rt.Run(); err != nil {
		panic(err.Error())
	}
}

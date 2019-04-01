package main

import (
	"context"
	"os"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/handlers/v1"
	"github.com/asecurityteam/runhttp"
	serverfull "github.com/asecurityteam/serverfull/pkg"
	serverfulldomain "github.com/asecurityteam/serverfull/pkg/domain"
	"github.com/asecurityteam/settings"
	"github.com/aws/aws-lambda-go/lambda"
)

type nopStorage struct{}

func (s *nopStorage) StoreCloudAsset(ctx context.Context, _ domain.CloudAssetChanges) error {
	runhttp.LoggerFromContext(ctx).Info("store cloud asset stub")
	return nil
}

func main() {
	ctx := context.Background()
	insert := &v1.CloudInsertHandler{
		LogFn:   runhttp.LoggerFromContext,
		StatFn:  runhttp.StatFromContext,
		Storage: &nopStorage{},
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

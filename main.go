package main

import (
	"context"
	"os"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	v1 "github.com/asecurityteam/asset-inventory-api/pkg/handlers/v1"
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
func (s *nopStorage) FetchCloudAsset(ctx context.Context, hostname string, ipAddress string, timestamp time.Time) (domain.CloudAssetDetails, error) {
	runhttp.LoggerFromContext(ctx).Info("fetch cloud asset stub")
	return domain.CloudAssetDetails{
		PrivateIPAddresses: []string{ipAddress},
		PublicIPAddresses:  []string{ipAddress},
		Hostnames:          []string{hostname},
		CreatedAt:          timestamp.Add(-1 * time.Second),
		DeletedAt:          timestamp.Add(1 * time.Second),
		Tags:               make(map[string]string),
	}, nil
}

func main() {
	ctx := context.Background()
	insert := &v1.CloudInsertHandler{
		LogFn:   runhttp.LoggerFromContext,
		StatFn:  runhttp.StatFromContext,
		Storage: &nopStorage{},
	}
	fetch := &v1.CloudFetchHandler{
		LogFn:   runhttp.LoggerFromContext,
		StatFn:  runhttp.StatFromContext,
		Storage: &nopStorage{},
	}
	handlers := map[string]serverfulldomain.Handler{
		"insert": lambda.NewHandler(insert.Handle),
		"fetch":  lambda.NewHandler(fetch.Handle),
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

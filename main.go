package main

import (
	"context"
	"os"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	v1 "github.com/asecurityteam/asset-inventory-api/pkg/handlers/v1"
	serverfull "github.com/asecurityteam/serverfull/pkg"
	serverfulldomain "github.com/asecurityteam/serverfull/pkg/domain"
	"github.com/asecurityteam/settings"
	"github.com/aws/aws-lambda-go/lambda"
)

type nopStorage struct{}

func (s *nopStorage) Store(ctx context.Context, _ domain.CloudAssetChanges) error {
	domain.LoggerFromContext(ctx).Info("store cloud asset stub")
	return nil
}

type nopFetcher struct{}

func (f *nopFetcher) FetchByIP(ctx context.Context, when time.Time, ipAddress string) ([]domain.CloudAssetDetails, error) {
	domain.LoggerFromContext(ctx).Info("fetch cloud asset by IP address stub")
	return []domain.CloudAssetDetails{
		{
			PrivateIPAddresses: []string{ipAddress},
			PublicIPAddresses:  []string{ipAddress},
		},
	}, nil
}

func (f *nopFetcher) FetchByHostname(ctx context.Context, when time.Time, hostname string) ([]domain.CloudAssetDetails, error) {
	domain.LoggerFromContext(ctx).Info("fetch cloud asset by hostname stub")
	return []domain.CloudAssetDetails{
		{
			Hostnames: []string{hostname},
			Tags: map[string]string{
				"Name": hostname,
			},
		},
	}, nil
}

func main() {
	ctx := context.Background()
	insert := &v1.CloudInsertHandler{
		LogFn:            domain.LoggerFromContext,
		StatFn:           domain.StatFromContext,
		CloudAssetStorer: &nopStorage{},
	}
	fetchByIP := &v1.CloudFetchByIPHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: &nopFetcher{},
	}
	fetchByHostname := &v1.CloudFetchByHostnameHandler{
		LogFn:   domain.LoggerFromContext,
		StatFn:  domain.StatFromContext,
		Fetcher: &nopFetcher{},
	}
	handlers := map[string]serverfulldomain.Handler{
		"insert":          lambda.NewHandler(insert.Handle),
		"fetchByIP":       lambda.NewHandler(fetchByIP.Handle),
		"fetchByHostname": lambda.NewHandler(fetchByHostname.Handle),
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

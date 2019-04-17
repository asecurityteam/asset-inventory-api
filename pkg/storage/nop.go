package storage

import (
	"context"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
)

// NopStorer is a nop implementation for the domain.CloudAssetStorer interface.
type NopStorer struct {
	LogFn domain.LogFn
}

// Store logs and returns a dummy response
func (s *NopStorer) Store(ctx context.Context, _ domain.CloudAssetChanges) error {
	s.LogFn(ctx).Info("store cloud asset stub")
	return nil
}

// NopFetcher is a nop implementation for the domain.CloudAssetByIPFetcher and
// domain.CloudAssetByHostnameFetcher interfaces.
type NopFetcher struct {
	LogFn domain.LogFn
}

// FetchByIP logs and returns a dummy response
func (f *NopFetcher) FetchByIP(ctx context.Context, when time.Time, ipAddress string) ([]domain.CloudAssetDetails, error) {
	f.LogFn(ctx).Info("fetch cloud asset by IP address stub")
	return []domain.CloudAssetDetails{
		{
			PrivateIPAddresses: []string{ipAddress},
			PublicIPAddresses:  []string{ipAddress},
		},
	}, nil
}

// FetchByHostname logs and returns a dummy response
func (f *NopFetcher) FetchByHostname(ctx context.Context, when time.Time, hostname string) ([]domain.CloudAssetDetails, error) {
	f.LogFn(ctx).Info("fetch cloud asset by hostname stub")
	return []domain.CloudAssetDetails{
		{
			Hostnames: []string{hostname},
			Tags: map[string]string{
				"Name": hostname,
			},
		},
	}, nil
}

package storage

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/logevent"
	"github.com/asecurityteam/runhttp"
	"github.com/stretchr/testify/assert"
)

func TestNopStorer(t *testing.T) {
	storer := &NopStorer{
		LogFn: func(_ context.Context) runhttp.Logger {
			return logevent.New(logevent.Config{Output: ioutil.Discard})
		},
	}

	e := storer.Store(context.Background(), domain.CloudAssetChanges{})
	assert.Nil(t, e)
}

func TestNopFetcherFetchByIP(t *testing.T) {
	fetcher := &NopFetcher{
		LogFn: func(_ context.Context) runhttp.Logger {
			return logevent.New(logevent.Config{Output: ioutil.Discard})
		},
	}
	ipAddress := "1.1.1.1"
	when := time.Now()
	output, e := fetcher.FetchByIP(context.Background(), when, ipAddress)
	assert.Nil(t, e)
	assert.Equal(t, ipAddress, output[0].PrivateIPAddresses[0])
	assert.Equal(t, ipAddress, output[0].PublicIPAddresses[0])
}

func TestNopFetcherFetchByHostname(t *testing.T) {
	fetcher := &NopFetcher{
		LogFn: func(_ context.Context) runhttp.Logger {
			return logevent.New(logevent.Config{Output: ioutil.Discard})
		},
	}
	hostname := "host1"
	when := time.Now()
	output, e := fetcher.FetchByHostname(context.Background(), when, hostname)
	assert.Nil(t, e)
	assert.Equal(t, hostname, output[0].Hostnames[0])
	assert.Equal(t, hostname, output[0].Tags["Name"])
}

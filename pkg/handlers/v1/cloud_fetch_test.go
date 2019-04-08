package v1

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/logevent"
	"github.com/asecurityteam/runhttp"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newFetchHandler(storage domain.Storage) *CloudFetchHandler {
	return &CloudFetchHandler{
		LogFn: func(_ context.Context) runhttp.Logger {
			return logevent.New(logevent.Config{Output: ioutil.Discard})
		},
		StatFn:  runhttp.StatFromContext,
		Storage: storage,
	}
}

func validFetchInput() CloudAssetFetchParameters {
	return CloudAssetFetchParameters{
		IPAddress: "1.1.1.1",
		Hostname:  "hostname",
		Timestamp: time.Now().Format(time.RFC3339Nano),
	}
}

func TestFetchInvalidInput(t *testing.T) {
	tc := []struct {
		name  string
		input CloudAssetFetchParameters
	}{
		{
			name:  "empty timestamp",
			input: CloudAssetFetchParameters{Timestamp: ""},
		},
		{
			name:  "invalid timestamp",
			input: CloudAssetFetchParameters{Timestamp: "foo"},
		},
		{
			name:  "no hostname or ipAddress",
			input: CloudAssetFetchParameters{Timestamp: time.Now().Format(time.RFC3339Nano)},
		},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(*testing.T) {
			_, e := newFetchHandler(nil).Handle(context.Background(), tt.input)
			assert.NotNil(t, e)

			_, ok := e.(InvalidInput)
			assert.True(t, ok)
		})
	}
}

func TestFetchStorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := NewMockStorage(ctrl)
	input := validFetchInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	storage.EXPECT().FetchCloudAsset(gomock.Any(), input.Hostname, input.IPAddress,
		ts).Return(domain.CloudAssetDetails{}, errors.New(""))

	_, e := newFetchHandler(storage).Handle(context.Background(), input)
	assert.NotNil(t, e)
}

func TestFetchStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := NewMockStorage(ctrl)
	input := validFetchInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	output := domain.CloudAssetDetails{
		PrivateIPAddresses: []string{input.IPAddress},
		PublicIPAddresses:  []string{input.IPAddress},
		Hostnames:          []string{input.Hostname},
		CreatedAt:          ts,
		DeletedAt:          ts,
	}
	storage.EXPECT().FetchCloudAsset(gomock.Any(), input.Hostname, input.IPAddress, ts).Return(output, nil)

	asset, e := newFetchHandler(storage).Handle(context.Background(), input)
	assert.Nil(t, e)
	assert.NotNil(t, asset)
}

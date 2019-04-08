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

func newInsertHandler(storage domain.Storage) *CloudInsertHandler {
	return &CloudInsertHandler{
		LogFn: func(_ context.Context) runhttp.Logger {
			return logevent.New(logevent.Config{Output: ioutil.Discard})
		},
		StatFn:  runhttp.StatFromContext,
		Storage: storage,
	}
}

func validInsertInput() CloudAssetChanges {
	return CloudAssetChanges{
		ChangeTime:   time.Now().Format(time.RFC3339Nano),
		ResourceID:   "cloud-resource-id",
		ResourceType: "cloud-resource-type",
		Region:       "cloud-region",
		AccountID:    "cloud-account-id",
		Tags:         make(map[string]string),
		Changes: []NetworkChanges{
			{
				PrivateIPAddresses: []string{"1.1.1.1"},
				PublicIPAddresses:  []string{"2.2.2.2"},
				Hostnames:          []string{"hostname"},
				ChangeType:         "ADDED",
			},
		},
	}
}

func TestInsertInvalidInput(t *testing.T) {
	input := CloudAssetChanges{
		ChangeTime: "not a timestamp",
	}
	e := newInsertHandler(nil).Handle(context.Background(), input)
	assert.NotNil(t, e)

	_, ok := e.(InvalidInput)
	assert.True(t, ok)
}

func TestInsertStorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := NewMockStorage(ctrl)
	storage.EXPECT().StoreCloudAsset(gomock.Any(), gomock.Any()).Return(errors.New(""))

	e := newInsertHandler(storage).Handle(context.Background(), validInsertInput())
	assert.NotNil(t, e)
}

func TestInsertStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := NewMockStorage(ctrl)
	storage.EXPECT().StoreCloudAsset(gomock.Any(), gomock.Any()).Return(nil)

	e := newInsertHandler(storage).Handle(context.Background(), validInsertInput())
	assert.Nil(t, e)
}

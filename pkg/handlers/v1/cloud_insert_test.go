package v1

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
)

func newInsertHandler(storer domain.CloudAssetStorer) *CloudInsertHandler {
	return &CloudInsertHandler{
		LogFn:            testLogFn,
		StatFn:           testStatFn,
		CloudAssetStorer: storer,
	}
}

func validInsertInput() CloudAssetChanges {
	return CloudAssetChanges{
		ChangeTime:   time.Now().Format(time.RFC3339Nano),
		ARN:          "cloud-resource-arn",
		ResourceType: "cloud-resource-type",
		Region:       "cloud-region",
		AccountID:    "cloud-account-id",
		Tags:         make(map[string]string),
		Changes: []NetworkChanges{
			{
				PrivateIPAddresses: []string{"1.1.1.1"},
				PublicIPAddresses:  []string{"2.2.2.2"},
				Hostnames:          []string{"hostname"},
				RelatedResources:   []string{"app/marketp-ALB-eeeeeee5555555/ffffffff66666666"},
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

	storage := NewMockCloudAssetStorer(ctrl)
	storage.EXPECT().Store(gomock.Any(), gomock.Any()).Return(errors.New(""))

	e := newInsertHandler(storage).Handle(context.Background(), validInsertInput())
	assert.NotNil(t, e)
}

func TestInsertStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := NewMockCloudAssetStorer(ctrl)
	storage.EXPECT().Store(gomock.Any(), gomock.Any()).Return(nil)

	e := newInsertHandler(storage).Handle(context.Background(), validInsertInput())
	assert.Nil(t, e)
}

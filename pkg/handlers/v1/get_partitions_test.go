package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestGetPartitionsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGetter := NewMockPartitionsGetter(ctrl)
	mockGetter.EXPECT().GetPartitions(gomock.Any()).Return(nil, errors.New(""))

	handler := GetPartitionsHandler{
		LogFn:  testLogFn,
		Getter: mockGetter,
	}

	_, err := handler.Handle(context.Background())
	assert.Error(t, err)
}

func TestGetPartitions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	partitions := []domain.Partition{
		{
			Name: "partitionA",
		},
		{
			Name: "partitionB",
		},
		{
			Name: "partitionC",
		},
	}

	mockGetter := NewMockPartitionsGetter(ctrl)
	mockGetter.EXPECT().GetPartitions(gomock.Any()).Return(partitions, nil)

	handler := GetPartitionsHandler{
		LogFn:  testLogFn,
		Getter: mockGetter,
	}

	results, err := handler.Handle(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, len(partitions), len(results.Results))
}

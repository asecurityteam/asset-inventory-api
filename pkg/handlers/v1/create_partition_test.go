package v1

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCreatePartitionNoTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerator := NewMockPartitionGenerator(ctrl)
	mockGenerator.EXPECT().GeneratePartition(gomock.Any(), time.Time{}, 0).Return(nil)

	h := &CreatePartitionHandler{
		LogFn:     testLogFn,
		Generator: mockGenerator,
	}

	err := h.Handle(context.Background(), CreatePartitionInput{})
	assert.NoError(t, err)
}

func TestCreatePartitionNoTimeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerator := NewMockPartitionGenerator(ctrl)
	mockGenerator.EXPECT().GeneratePartition(gomock.Any(), time.Time{}, 0).Return(errors.New(""))

	h := &CreatePartitionHandler{
		LogFn:     testLogFn,
		Generator: mockGenerator,
	}

	err := h.Handle(context.Background(), CreatePartitionInput{})
	assert.Error(t, err)
}

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
	mockGenerator.EXPECT().GeneratePartition(gomock.Any()).Return(nil)

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
	mockGenerator.EXPECT().GeneratePartition(gomock.Any()).Return(errors.New(""))

	h := &CreatePartitionHandler{
		LogFn:     testLogFn,
		Generator: mockGenerator,
	}

	err := h.Handle(context.Background(), CreatePartitionInput{})
	assert.Error(t, err)
}

func TestPartitionTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerator := NewMockPartitionGenerator(ctrl)
	mockGenerator.EXPECT().GeneratePartitionWithTimestamp(gomock.Any(), gomock.Any()).Return(nil)

	h := &CreatePartitionHandler{
		LogFn:     testLogFn,
		Generator: mockGenerator,
	}

	ts := time.Now().Format(time.RFC3339)

	err := h.Handle(context.Background(), CreatePartitionInput{Timestamp: ts})
	assert.NoError(t, err)
}

func TestPartitionTimeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerator := NewMockPartitionGenerator(ctrl)
	mockGenerator.EXPECT().GeneratePartitionWithTimestamp(gomock.Any(), gomock.Any()).Return(errors.New(""))

	h := &CreatePartitionHandler{
		LogFn:     testLogFn,
		Generator: mockGenerator,
	}

	ts := time.Now().Format(time.RFC3339)

	err := h.Handle(context.Background(), CreatePartitionInput{Timestamp: ts})
	assert.Error(t, err)
}

func TestPartitionTimeBadTimestamp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &CreatePartitionHandler{
		LogFn: testLogFn,
	}

	ts := "not a valid ts"

	err := h.Handle(context.Background(), CreatePartitionInput{Timestamp: ts})
	assert.Error(t, err)
}

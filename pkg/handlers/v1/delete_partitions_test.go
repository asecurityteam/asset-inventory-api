package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDeletePartitions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	paritionName := "TEST"

	mockDeleter := NewMockPartitionsDeleter(ctrl)
	mockDeleter.EXPECT().DeletePartitions(gomock.Any(), paritionName).Return(nil)

	h := &DeletePartitionsHandler{
		LogFn:   testLogFn,
		Deleter: mockDeleter,
	}

	err := h.Handle(context.Background(), DeletePartitionsInput{Name: paritionName})
	assert.NoError(t, err)
}

func TestDeletePartitionsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	partitionName := "TEST"

	mockDeleter := NewMockPartitionsDeleter(ctrl)
	mockDeleter.EXPECT().DeletePartitions(gomock.Any(), partitionName).Return(errors.New(""))

	h := &DeletePartitionsHandler{
		LogFn:   testLogFn,
		Deleter: mockDeleter,
	}

	err := h.Handle(context.Background(), DeletePartitionsInput{Name: partitionName})
	assert.Error(t, err)
}

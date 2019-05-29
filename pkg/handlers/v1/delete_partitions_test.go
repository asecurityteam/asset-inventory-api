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

	days := 365

	mockDeleter := NewMockPartitionsDeleter(ctrl)
	mockDeleter.EXPECT().DeletePartitions(gomock.Any(), days).Return(10, nil)

	h := &DeletePartitionsHandler{
		LogFn:   testLogFn,
		Deleter: mockDeleter,
	}

	res, err := h.Handle(context.Background(), DeletePartitionsInput{Days: days})
	assert.NoError(t, err)
	assert.Equal(t, 10, res.Deleted)
}

func TestDeletePartitionsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	days := 365

	mockDeleter := NewMockPartitionsDeleter(ctrl)
	mockDeleter.EXPECT().DeletePartitions(gomock.Any(), days).Return(0, errors.New(""))

	h := &DeletePartitionsHandler{
		LogFn:   testLogFn,
		Deleter: mockDeleter,
	}

	_, err := h.Handle(context.Background(), DeletePartitionsInput{Days: days})
	assert.Error(t, err)
}

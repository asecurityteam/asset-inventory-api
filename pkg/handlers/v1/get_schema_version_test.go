package v1

import (
	"context"
	"errors"
	"math/rand"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestGetSchemaVersionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGetter := NewMockSchemaVersionGetter(ctrl)
	mockGetter.EXPECT().GetSchemaVersion(gomock.Any()).Return(uint(0), errors.New(""))

	handler := GetSchemaVersionHandler{
		LogFn:  testLogFn,
		Getter: mockGetter,
	}

	_, err := handler.Handle(context.Background())
	assert.Error(t, err)
}

func TestGetSchemaVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := uint(rand.Uint32() + 1) // version number to use for test

	mockGetter := NewMockSchemaVersionGetter(ctrl)
	mockGetter.EXPECT().GetSchemaVersion(gomock.Any()).Return(v, nil)

	handler := GetSchemaVersionHandler{
		LogFn:  testLogFn,
		Getter: mockGetter,
	}

	res, err := handler.Handle(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, v, res.Version)
}

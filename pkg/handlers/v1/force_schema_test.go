package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestForceSchemaHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockForcer := NewMockSchemaVersionForcer(ctrl)
	mockForcer.EXPECT().ForceSchemaToVersion(gomock.Any(), gomock.Any()).Return(nil)

	handler := ForceSchemaHandler{
		LogFn:               testLogFn,
		SchemaVersionForcer: mockForcer,
	}

	err := handler.Handle(context.Background(), SchemaVersion{Version: 1})
	assert.NoError(t, err)
}

func TestForceSchemaHandlerErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockForcer := NewMockSchemaVersionForcer(ctrl)
	mockForcer.EXPECT().ForceSchemaToVersion(gomock.Any(), gomock.Any()).Return(errors.New("error forcing schema version"))

	handler := ForceSchemaHandler{
		LogFn:               testLogFn,
		SchemaVersionForcer: mockForcer,
	}

	err := handler.Handle(context.Background(), SchemaVersion{Version: 1})
	assert.Error(t, err)
}

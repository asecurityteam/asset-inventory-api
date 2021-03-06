package v1

import (
	"context"
	"errors"
	"math/rand"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestSchemaVersionStepUpError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := NewMockSchemaMigratorUp(ctrl)
	mockMigrator.EXPECT().MigrateSchemaUp(gomock.Any()).Return(uint(0), errors.New(""))

	handler := SchemaVersionStepUpHandler{
		LogFn:    testLogFn,
		Migrator: mockMigrator,
	}

	_, err := handler.Handle(context.Background())
	assert.Error(t, err)
}

func TestSchemaVersionStepUp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := uint(rand.Uint32() + 1) // version number to use for test

	mockMigrator := NewMockSchemaMigratorUp(ctrl)
	mockMigrator.EXPECT().MigrateSchemaUp(gomock.Any()).Return(v, nil)

	handler := SchemaVersionStepUpHandler{
		LogFn:    testLogFn,
		Migrator: mockMigrator,
	}

	res, err := handler.Handle(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, v, res.Version)
}

package v1

import (
	"context"
	"errors"
	"math/rand"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestSchemaVersionDownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := NewMockSchemaMigratorDown(ctrl)
	mockMigrator.EXPECT().MigrateSchemaDown(gomock.Any()).Return(uint(0), errors.New(""))

	handler := SchemaVersionDownHandler{
		LogFn:    testLogFn,
		Migrator: mockMigrator,
	}

	_, err := handler.Handle(context.Background())
	assert.Error(t, err)
}

func TestSchemaVersionDown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := uint(rand.Uint32() + 1) // version number to use for test

	mockMigrator := NewMockSchemaMigratorDown(ctrl)
	mockMigrator.EXPECT().MigrateSchemaDown(gomock.Any()).Return(v, nil)

	handler := SchemaVersionDownHandler{
		LogFn:    testLogFn,
		Migrator: mockMigrator,
	}

	res, err := handler.Handle(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, v, res.Version)
}

package storage

import (
	"context"
	"errors"
	"math/rand"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestGetSchemaVersionErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockStorageSchemaMigrator(ctrl)
	migrator.EXPECT().Version().Return(uint(0), false, errors.New("something went wrong"))
	sm := &SchemaManager{migrator: migrator}
	_, err := sm.GetSchemaVersion(context.Background())
	require.Error(t, err)
}

func TestGetSchemaVersionNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockStorageSchemaMigrator(ctrl)
	migrator.EXPECT().Version().Return(uint(0), false, migrate.ErrNilVersion)
	migrator.EXPECT().Version().Return(uint(0), false, migrate.ErrNilVersion)
	sm := &SchemaManager{migrator: migrator}
	v, err := sm.GetSchemaVersion(context.Background())
	require.Equal(t, uint(0), v)
	require.Nil(t, err)
}

func TestGetSchemaVersionSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	version := uint(rand.Uint64())
	migrator := NewMockStorageSchemaMigrator(ctrl)
	migrator.EXPECT().Version().Return(version, false, nil)
	migrator.EXPECT().Version().Return(version, false, nil)
	sm := &SchemaManager{migrator: migrator}
	v, err := sm.GetSchemaVersion(context.Background())
	require.Equal(t, version, v)
	require.Nil(t, err)
}

func TestForceSchemaToVersionErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	version := uint(rand.Uint64())
	migrator := NewMockStorageSchemaMigrator(ctrl)
	migrator.EXPECT().Version().Return(version, false, nil)
	migrator.EXPECT().Force(int(version)).Return(errors.New("something happened"))
	sm := &SchemaManager{migrator: migrator}
	err := sm.ForceSchemaToVersion(context.Background(), version)
	require.Error(t, err)
}

func TestForceSchemaToVersionSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	version := uint(rand.Uint64())
	migrator := NewMockStorageSchemaMigrator(ctrl)
	migrator.EXPECT().Version().Return(version, false, nil)
	migrator.EXPECT().Force(int(version)).Return(nil)
	sm := &SchemaManager{migrator: migrator}
	err := sm.ForceSchemaToVersion(context.Background(), version)
	require.Nil(t, err)
}

func TestMigrateSchemaToVersionErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	version := uint(rand.Uint64())
	migrator := NewMockStorageSchemaMigrator(ctrl)
	migrator.EXPECT().Version().Return(version, false, nil)
	migrator.EXPECT().Migrate(version).Return(errors.New("something happened"))
	sm := &SchemaManager{migrator: migrator}
	err := sm.MigrateSchemaToVersion(context.Background(), version)
	require.Error(t, err)
}

func TestMigrateSchemaToVersionSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	version := uint(rand.Uint64())
	migrator := NewMockStorageSchemaMigrator(ctrl)
	migrator.EXPECT().Version().Return(version, false, nil)
	migrator.EXPECT().Migrate(version).Return(nil)
	sm := &SchemaManager{migrator: migrator}
	err := sm.MigrateSchemaToVersion(context.Background(), version)
	require.Nil(t, err)
}

func TestMigrateSchemaUpErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockStorageSchemaMigrator(ctrl)
	migrator.EXPECT().Version().Return(uint(0), false, nil)
	migrator.EXPECT().Steps(gomock.Any()).Return(errors.New("something happened"))
	sm := &SchemaManager{migrator: migrator}
	_, err := sm.MigrateSchemaUp(context.Background())
	require.Error(t, err)
}

func TestMigrateSchemaUpSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	version := uint(rand.Uint64())
	migrator := NewMockStorageSchemaMigrator(ctrl)
	migrator.EXPECT().Version().Return(uint(0), false, nil)
	migrator.EXPECT().Version().Return(uint(0), false, nil)
	migrator.EXPECT().Steps(gomock.Any()).Return(nil)
	migrator.EXPECT().Version().Return(version, false, nil) //Version() (version uint, dirty bool, err error)
	sm := &SchemaManager{migrator: migrator}
	v, err := sm.MigrateSchemaUp(context.Background())
	require.NoError(t, err)
	require.Equal(t, version, v)
}

func TestMigrateSchemaDownErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockStorageSchemaMigrator(ctrl)
	migrator.EXPECT().Version().Return(uint(0), false, nil)
	migrator.EXPECT().Steps(gomock.Any()).Return(errors.New("something happened"))
	sm := &SchemaManager{migrator: migrator}
	_, err := sm.MigrateSchemaDown(context.Background())
	require.Error(t, err)
}

func TestMigrateSchemaDownSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	version := uint(rand.Uint64())
	migrator := NewMockStorageSchemaMigrator(ctrl)
	migrator.EXPECT().Version().Return(uint(0), false, nil)
	migrator.EXPECT().Version().Return(uint(0), false, nil)
	migrator.EXPECT().Steps(gomock.Any()).Return(nil)
	migrator.EXPECT().Version().Return(version, false, nil) //Version() (version uint, dirty bool, err error)
	sm := &SchemaManager{migrator: migrator}
	v, err := sm.MigrateSchemaDown(context.Background())
	require.NoError(t, err)
	require.Equal(t, version, v)
}

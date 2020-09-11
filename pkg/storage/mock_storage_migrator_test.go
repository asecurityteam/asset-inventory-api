// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/asecurityteam/asset-inventory-api/pkg/domain (interfaces: StorageSchemaMigrator)

// Package storage is a generated GoMock package.
package storage

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockStorageSchemaMigrator is a mock of StorageSchemaMigrator interface
type MockStorageSchemaMigrator struct {
	ctrl     *gomock.Controller
	recorder *MockStorageSchemaMigratorMockRecorder
}

// MockStorageSchemaMigratorMockRecorder is the mock recorder for MockStorageSchemaMigrator
type MockStorageSchemaMigratorMockRecorder struct {
	mock *MockStorageSchemaMigrator
}

// NewMockStorageSchemaMigrator creates a new mock instance
func NewMockStorageSchemaMigrator(ctrl *gomock.Controller) *MockStorageSchemaMigrator {
	mock := &MockStorageSchemaMigrator{ctrl: ctrl}
	mock.recorder = &MockStorageSchemaMigratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockStorageSchemaMigrator) EXPECT() *MockStorageSchemaMigratorMockRecorder {
	return m.recorder
}

// Close mocks base method
func (m *MockStorageSchemaMigrator) Close() (error, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Close indicates an expected call of Close
func (mr *MockStorageSchemaMigratorMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockStorageSchemaMigrator)(nil).Close))
}

// Force mocks base method
func (m *MockStorageSchemaMigrator) Force(arg0 int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Force", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Force indicates an expected call of Force
func (mr *MockStorageSchemaMigratorMockRecorder) Force(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Force", reflect.TypeOf((*MockStorageSchemaMigrator)(nil).Force), arg0)
}

// Migrate mocks base method
func (m *MockStorageSchemaMigrator) Migrate(arg0 uint) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Migrate", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Migrate indicates an expected call of Migrate
func (mr *MockStorageSchemaMigratorMockRecorder) Migrate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Migrate", reflect.TypeOf((*MockStorageSchemaMigrator)(nil).Migrate), arg0)
}

// Steps mocks base method
func (m *MockStorageSchemaMigrator) Steps(arg0 int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Steps", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Steps indicates an expected call of Steps
func (mr *MockStorageSchemaMigratorMockRecorder) Steps(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Steps", reflect.TypeOf((*MockStorageSchemaMigrator)(nil).Steps), arg0)
}

// Version mocks base method
func (m *MockStorageSchemaMigrator) Version() (uint, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Version")
	ret0, _ := ret[0].(uint)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Version indicates an expected call of Version
func (mr *MockStorageSchemaMigratorMockRecorder) Version() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Version", reflect.TypeOf((*MockStorageSchemaMigrator)(nil).Version))
}

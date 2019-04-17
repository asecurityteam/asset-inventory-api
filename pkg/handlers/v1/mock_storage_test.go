// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/domain/storage.go

// Package v1 is a generated GoMock package.
package v1

import (
	context "context"
	domain "github.com/asecurityteam/asset-inventory-api/pkg/domain"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
	time "time"
)

// MockCloudAssetStorer is a mock of CloudAssetStorer interface
type MockCloudAssetStorer struct {
	ctrl     *gomock.Controller
	recorder *MockCloudAssetStorerMockRecorder
}

// MockCloudAssetStorerMockRecorder is the mock recorder for MockCloudAssetStorer
type MockCloudAssetStorerMockRecorder struct {
	mock *MockCloudAssetStorer
}

// NewMockCloudAssetStorer creates a new mock instance
func NewMockCloudAssetStorer(ctrl *gomock.Controller) *MockCloudAssetStorer {
	mock := &MockCloudAssetStorer{ctrl: ctrl}
	mock.recorder = &MockCloudAssetStorerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCloudAssetStorer) EXPECT() *MockCloudAssetStorerMockRecorder {
	return m.recorder
}

// Store mocks base method
func (m *MockCloudAssetStorer) Store(arg0 context.Context, arg1 domain.CloudAssetChanges) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Store", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Store indicates an expected call of Store
func (mr *MockCloudAssetStorerMockRecorder) Store(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Store", reflect.TypeOf((*MockCloudAssetStorer)(nil).Store), arg0, arg1)
}

// MockCloudAssetByIPFetcher is a mock of CloudAssetByIPFetcher interface
type MockCloudAssetByIPFetcher struct {
	ctrl     *gomock.Controller
	recorder *MockCloudAssetByIPFetcherMockRecorder
}

// MockCloudAssetByIPFetcherMockRecorder is the mock recorder for MockCloudAssetByIPFetcher
type MockCloudAssetByIPFetcherMockRecorder struct {
	mock *MockCloudAssetByIPFetcher
}

// NewMockCloudAssetByIPFetcher creates a new mock instance
func NewMockCloudAssetByIPFetcher(ctrl *gomock.Controller) *MockCloudAssetByIPFetcher {
	mock := &MockCloudAssetByIPFetcher{ctrl: ctrl}
	mock.recorder = &MockCloudAssetByIPFetcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCloudAssetByIPFetcher) EXPECT() *MockCloudAssetByIPFetcherMockRecorder {
	return m.recorder
}

// FetchByIP mocks base method
func (m *MockCloudAssetByIPFetcher) FetchByIP(ctx context.Context, when time.Time, ipAddress string) ([]domain.CloudAssetDetails, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchByIP", ctx, when, ipAddress)
	ret0, _ := ret[0].([]domain.CloudAssetDetails)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchByIP indicates an expected call of FetchByIP
func (mr *MockCloudAssetByIPFetcherMockRecorder) FetchByIP(ctx, when, ipAddress interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchByIP", reflect.TypeOf((*MockCloudAssetByIPFetcher)(nil).FetchByIP), ctx, when, ipAddress)
}

// MockCloudAssetByHostnameFetcher is a mock of CloudAssetByHostnameFetcher interface
type MockCloudAssetByHostnameFetcher struct {
	ctrl     *gomock.Controller
	recorder *MockCloudAssetByHostnameFetcherMockRecorder
}

// MockCloudAssetByHostnameFetcherMockRecorder is the mock recorder for MockCloudAssetByHostnameFetcher
type MockCloudAssetByHostnameFetcherMockRecorder struct {
	mock *MockCloudAssetByHostnameFetcher
}

// NewMockCloudAssetByHostnameFetcher creates a new mock instance
func NewMockCloudAssetByHostnameFetcher(ctrl *gomock.Controller) *MockCloudAssetByHostnameFetcher {
	mock := &MockCloudAssetByHostnameFetcher{ctrl: ctrl}
	mock.recorder = &MockCloudAssetByHostnameFetcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCloudAssetByHostnameFetcher) EXPECT() *MockCloudAssetByHostnameFetcherMockRecorder {
	return m.recorder
}

// FetchByHostname mocks base method
func (m *MockCloudAssetByHostnameFetcher) FetchByHostname(ctx context.Context, when time.Time, hostname string) ([]domain.CloudAssetDetails, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchByHostname", ctx, when, hostname)
	ret0, _ := ret[0].([]domain.CloudAssetDetails)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchByHostname indicates an expected call of FetchByHostname
func (mr *MockCloudAssetByHostnameFetcherMockRecorder) FetchByHostname(ctx, when, hostname interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchByHostname", reflect.TypeOf((*MockCloudAssetByHostnameFetcher)(nil).FetchByHostname), ctx, when, hostname)
}

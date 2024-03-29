// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/backends/backend.go

// Package backends is a generated GoMock package.
package backends

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	zap "go.uber.org/zap"
	kubernetes "k8s.io/client-go/kubernetes"
	v10 "k8s.io/client-go/listers/core/v1"

	v1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	versioned "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

// MockBackend is a mock of Backend interface.
type MockBackend struct {
	ctrl     *gomock.Controller
	recorder *MockBackendMockRecorder
}

// MockBackendMockRecorder is the mock recorder for MockBackend.
type MockBackendMockRecorder struct {
	mock *MockBackend
}

// NewMockBackend creates a new mock instance.
func NewMockBackend(ctrl *gomock.Controller) *MockBackend {
	mock := &MockBackend{ctrl: ctrl}
	mock.recorder = &MockBackendMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackend) EXPECT() *MockBackendMockRecorder {
	return m.recorder
}

// Sync mocks base method.
func (m *MockBackend) Sync(ctx context.Context, loadTest v1.LoadTest, reportURL string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Sync", ctx, loadTest, reportURL)
	ret0, _ := ret[0].(error)
	return ret0
}

// Sync indicates an expected call of Sync.
func (mr *MockBackendMockRecorder) Sync(ctx, loadTest, reportURL interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Sync", reflect.TypeOf((*MockBackend)(nil).Sync), ctx, loadTest, reportURL)
}

// SyncStatus mocks base method.
func (m *MockBackend) SyncStatus(ctx context.Context, loadTest v1.LoadTest, loadTestStatus *v1.LoadTestStatus) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SyncStatus", ctx, loadTest, loadTestStatus)
	ret0, _ := ret[0].(error)
	return ret0
}

// SyncStatus indicates an expected call of SyncStatus.
func (mr *MockBackendMockRecorder) SyncStatus(ctx, loadTest, loadTestStatus interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SyncStatus", reflect.TypeOf((*MockBackend)(nil).SyncStatus), ctx, loadTest, loadTestStatus)
}

// TransformLoadTestSpec mocks base method.
func (m *MockBackend) TransformLoadTestSpec(spec *v1.LoadTestSpec) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TransformLoadTestSpec", spec)
	ret0, _ := ret[0].(error)
	return ret0
}

// TransformLoadTestSpec indicates an expected call of TransformLoadTestSpec.
func (mr *MockBackendMockRecorder) TransformLoadTestSpec(spec interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TransformLoadTestSpec", reflect.TypeOf((*MockBackend)(nil).TransformLoadTestSpec), spec)
}

// Type mocks base method.
func (m *MockBackend) Type() v1.LoadTestType {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Type")
	ret0, _ := ret[0].(v1.LoadTestType)
	return ret0
}

// Type indicates an expected call of Type.
func (mr *MockBackendMockRecorder) Type() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Type", reflect.TypeOf((*MockBackend)(nil).Type))
}

// GetEnvConfig mocks base method.
func (m *MockBackend) GetEnvConfig() interface{} {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEnvConfig")
	ret0, _ := ret[0].(interface{})
	return ret0
}

// GetEnvConfig indicates an expected call of GetEnvConfig.
func (mr *MockBackendMockRecorder) GetEnvConfig() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEnvConfig", reflect.TypeOf((*MockBackend)(nil).GetEnvConfig))
}

// SetDefaults mocks base method.
func (m *MockBackend) SetDefaults() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetDefaults")
}

// SetDefaults indicates an expected call of SetDefaults.
func (mr *MockBackendMockRecorder) SetDefaults() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDefaults", reflect.TypeOf((*MockBackend)(nil).SetDefaults))
}


// SetLogger mocks base method.
func (m *MockBackend) SetLogger(arg0 *zap.Logger) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetLogger", arg0)
}

// SetLogger indicates an expected call of SetLogger.
func (mr *MockBackendMockRecorder) SetLogger(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetLogger", reflect.TypeOf((*MockBackend)(nil).SetLogger), arg0)
}


// SetPodAnnotations mocks base method.
func (m *MockBackend) SetPodAnnotations(arg0 map[string]string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetPodAnnotations", arg0)
}

// SetPodAnnotations indicates an expected call of SetPodAnnotations.
func (mr *MockBackendMockRecorder) SetPodAnnotations(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetPodAnnotations", reflect.TypeOf((*MockBackend)(nil).SetPodAnnotations), arg0)
}

// SetPodNodeSelector mocks base method.
func (m *MockBackend) SetPodNodeSelector(arg0 map[string]string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetPodNodeSelector", arg0)
}

// SetPodNodeSelector indicates an expected call of SetPodNodeSelector.
func (mr *MockBackendMockRecorder) SetPodNodeSelector(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetPodNodeSelector", reflect.TypeOf((*MockBackend)(nil).SetPodNodeSelector), arg0)
}

// SetKubeClientSet mocks base method.
func (m *MockBackend) SetKubeClientSet(arg0 kubernetes.Interface) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetKubeClientSet", arg0)
}

// SetKubeClientSet indicates an expected call of SetKubeClientSet.
func (mr *MockBackendMockRecorder) SetKubeClientSet(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetKubeClientSet", reflect.TypeOf((*MockBackend)(nil).SetKubeClientSet), arg0)
}

// SetKangalClientSet mocks base method.
func (m *MockBackend) SetKangalClientSet(arg0 versioned.Interface) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetKangalClientSet", arg0)
}

// SetKangalClientSet indicates an expected call of SetKangalClientSet.
func (mr *MockBackendMockRecorder) SetKangalClientSet(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetKangalClientSet", reflect.TypeOf((*MockBackend)(nil).SetKangalClientSet), arg0)
}

// SetNamespaceLister mocks base method.
func (m *MockBackend) SetNamespaceLister(arg0 v10.NamespaceLister) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetNamespaceLister", arg0)
}

// SetNamespaceLister indicates an expected call of SetNamespaceLister.
func (mr *MockBackendMockRecorder) SetNamespaceLister(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetNamespaceLister", reflect.TypeOf((*MockBackend)(nil).SetNamespaceLister), arg0)
}

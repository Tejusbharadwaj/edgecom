// Code generated by MockGen. DO NOT EDIT.
// Source: ../../proto/timeseries_grpc.pb.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	proto "github.com/tejusbharadwaj/edgecom/proto"
	grpc "google.golang.org/grpc"
)

// MockTimeSeriesServiceClient is a mock of TimeSeriesServiceClient interface.
type MockTimeSeriesServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockTimeSeriesServiceClientMockRecorder
}

// MockTimeSeriesServiceClientMockRecorder is the mock recorder for MockTimeSeriesServiceClient.
type MockTimeSeriesServiceClientMockRecorder struct {
	mock *MockTimeSeriesServiceClient
}

// NewMockTimeSeriesServiceClient creates a new mock instance.
func NewMockTimeSeriesServiceClient(ctrl *gomock.Controller) *MockTimeSeriesServiceClient {
	mock := &MockTimeSeriesServiceClient{ctrl: ctrl}
	mock.recorder = &MockTimeSeriesServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTimeSeriesServiceClient) EXPECT() *MockTimeSeriesServiceClientMockRecorder {
	return m.recorder
}

// QueryTimeSeries mocks base method.
func (m *MockTimeSeriesServiceClient) QueryTimeSeries(ctx context.Context, in *proto.TimeSeriesRequest, opts ...grpc.CallOption) (*proto.TimeSeriesResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "QueryTimeSeries", varargs...)
	ret0, _ := ret[0].(*proto.TimeSeriesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryTimeSeries indicates an expected call of QueryTimeSeries.
func (mr *MockTimeSeriesServiceClientMockRecorder) QueryTimeSeries(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryTimeSeries", reflect.TypeOf((*MockTimeSeriesServiceClient)(nil).QueryTimeSeries), varargs...)
}

// MockTimeSeriesServiceServer is a mock of TimeSeriesServiceServer interface.
type MockTimeSeriesServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockTimeSeriesServiceServerMockRecorder
}

// MockTimeSeriesServiceServerMockRecorder is the mock recorder for MockTimeSeriesServiceServer.
type MockTimeSeriesServiceServerMockRecorder struct {
	mock *MockTimeSeriesServiceServer
}

// NewMockTimeSeriesServiceServer creates a new mock instance.
func NewMockTimeSeriesServiceServer(ctrl *gomock.Controller) *MockTimeSeriesServiceServer {
	mock := &MockTimeSeriesServiceServer{ctrl: ctrl}
	mock.recorder = &MockTimeSeriesServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTimeSeriesServiceServer) EXPECT() *MockTimeSeriesServiceServerMockRecorder {
	return m.recorder
}

// QueryTimeSeries mocks base method.
func (m *MockTimeSeriesServiceServer) QueryTimeSeries(arg0 context.Context, arg1 *proto.TimeSeriesRequest) (*proto.TimeSeriesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryTimeSeries", arg0, arg1)
	ret0, _ := ret[0].(*proto.TimeSeriesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryTimeSeries indicates an expected call of QueryTimeSeries.
func (mr *MockTimeSeriesServiceServerMockRecorder) QueryTimeSeries(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryTimeSeries", reflect.TypeOf((*MockTimeSeriesServiceServer)(nil).QueryTimeSeries), arg0, arg1)
}

// mustEmbedUnimplementedTimeSeriesServiceServer mocks base method.
func (m *MockTimeSeriesServiceServer) mustEmbedUnimplementedTimeSeriesServiceServer() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "mustEmbedUnimplementedTimeSeriesServiceServer")
}

// mustEmbedUnimplementedTimeSeriesServiceServer indicates an expected call of mustEmbedUnimplementedTimeSeriesServiceServer.
func (mr *MockTimeSeriesServiceServerMockRecorder) mustEmbedUnimplementedTimeSeriesServiceServer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "mustEmbedUnimplementedTimeSeriesServiceServer", reflect.TypeOf((*MockTimeSeriesServiceServer)(nil).mustEmbedUnimplementedTimeSeriesServiceServer))
}

// MockUnsafeTimeSeriesServiceServer is a mock of UnsafeTimeSeriesServiceServer interface.
type MockUnsafeTimeSeriesServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockUnsafeTimeSeriesServiceServerMockRecorder
}

// MockUnsafeTimeSeriesServiceServerMockRecorder is the mock recorder for MockUnsafeTimeSeriesServiceServer.
type MockUnsafeTimeSeriesServiceServerMockRecorder struct {
	mock *MockUnsafeTimeSeriesServiceServer
}

// NewMockUnsafeTimeSeriesServiceServer creates a new mock instance.
func NewMockUnsafeTimeSeriesServiceServer(ctrl *gomock.Controller) *MockUnsafeTimeSeriesServiceServer {
	mock := &MockUnsafeTimeSeriesServiceServer{ctrl: ctrl}
	mock.recorder = &MockUnsafeTimeSeriesServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUnsafeTimeSeriesServiceServer) EXPECT() *MockUnsafeTimeSeriesServiceServerMockRecorder {
	return m.recorder
}

// mustEmbedUnimplementedTimeSeriesServiceServer mocks base method.
func (m *MockUnsafeTimeSeriesServiceServer) mustEmbedUnimplementedTimeSeriesServiceServer() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "mustEmbedUnimplementedTimeSeriesServiceServer")
}

// mustEmbedUnimplementedTimeSeriesServiceServer indicates an expected call of mustEmbedUnimplementedTimeSeriesServiceServer.
func (mr *MockUnsafeTimeSeriesServiceServerMockRecorder) mustEmbedUnimplementedTimeSeriesServiceServer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "mustEmbedUnimplementedTimeSeriesServiceServer", reflect.TypeOf((*MockUnsafeTimeSeriesServiceServer)(nil).mustEmbedUnimplementedTimeSeriesServiceServer))
}

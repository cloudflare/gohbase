// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/tsuna/gohbase/hrpc (interfaces: Call)

package mock

import (
	gomock "github.com/golang/mock/gomock"
	proto "github.com/golang/protobuf/proto"
	filter "github.com/tsuna/gohbase/filter"
	hrpc "github.com/tsuna/gohbase/hrpc"
	context "context"
)

// Mock of Call interface
type MockCall struct {
	ctrl     *gomock.Controller
	recorder *_MockCallRecorder
}

// Recorder for MockCall (not exported)
type _MockCallRecorder struct {
	mock *MockCall
}

func NewMockCall(ctrl *gomock.Controller) *MockCall {
	mock := &MockCall{ctrl: ctrl}
	mock.recorder = &_MockCallRecorder{mock}
	return mock
}

func (_m *MockCall) EXPECT() *_MockCallRecorder {
	return _m.recorder
}

func (_m *MockCall) Context() context.Context {
	ret := _m.ctrl.Call(_m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

func (_mr *_MockCallRecorder) Context() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Context")
}

func (_m *MockCall) Key() []byte {
	ret := _m.ctrl.Call(_m, "Key")
	ret0, _ := ret[0].([]byte)
	return ret0
}

func (_mr *_MockCallRecorder) Key() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Key")
}

func (_m *MockCall) Name() string {
	ret := _m.ctrl.Call(_m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

func (_mr *_MockCallRecorder) Name() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Name")
}

func (_m *MockCall) NewResponse() proto.Message {
	ret := _m.ctrl.Call(_m, "NewResponse")
	ret0, _ := ret[0].(proto.Message)
	return ret0
}

func (_mr *_MockCallRecorder) NewResponse() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "NewResponse")
}

func (_m *MockCall) Region() hrpc.RegionInfo {
	ret := _m.ctrl.Call(_m, "Region")
	ret0, _ := ret[0].(hrpc.RegionInfo)
	return ret0
}

func (_mr *_MockCallRecorder) Region() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Region")
}

func (_m *MockCall) ResultChan() chan hrpc.RPCResult {
	ret := _m.ctrl.Call(_m, "ResultChan")
	ret0, _ := ret[0].(chan hrpc.RPCResult)
	return ret0
}

func (_mr *_MockCallRecorder) ResultChan() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "ResultChan")
}

func (_m *MockCall) Serialize() ([]byte, error) {
	ret := _m.ctrl.Call(_m, "Serialize")
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockCallRecorder) Serialize() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Serialize")
}

func (_m *MockCall) SetFamilies(_param0 map[string][]string) error {
	ret := _m.ctrl.Call(_m, "SetFamilies", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockCallRecorder) SetFamilies(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SetFamilies", arg0)
}

func (_m *MockCall) SetFilter(_param0 filter.Filter) error {
	ret := _m.ctrl.Call(_m, "SetFilter", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockCallRecorder) SetFilter(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SetFilter", arg0)
}

func (_m *MockCall) SetRegion(_param0 hrpc.RegionInfo) {
	_m.ctrl.Call(_m, "SetRegion", _param0)
}

func (_mr *_MockCallRecorder) SetRegion(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SetRegion", arg0)
}

func (_m *MockCall) Table() []byte {
	ret := _m.ctrl.Call(_m, "Table")
	ret0, _ := ret[0].([]byte)
	return ret0
}

func (_mr *_MockCallRecorder) Table() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Table")
}

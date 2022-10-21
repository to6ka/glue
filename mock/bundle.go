// Code generated by mockery v2.14.0. DO NOT EDIT.

package mock

import (
	di "github.com/gozix/di"

	mock "github.com/stretchr/testify/mock"
)

// Bundle is an autogenerated mock type for the Bundle type
type Bundle struct {
	mock.Mock
}

// Build provides a mock function with given fields: builder
func (_m *Bundle) Build(builder di.Builder) error {
	ret := _m.Called(builder)

	var r0 error
	if rf, ok := ret.Get(0).(func(di.Builder) error); ok {
		r0 = rf(builder)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Name provides a mock function with given fields:
func (_m *Bundle) Name() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

type mockConstructorTestingTNewBundle interface {
	mock.TestingT
	Cleanup(func())
}

// NewBundle creates a new instance of Bundle. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBundle(t mockConstructorTestingTNewBundle) *Bundle {
	mock := &Bundle{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

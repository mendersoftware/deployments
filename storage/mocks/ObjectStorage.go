// Copyright 2022 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

// Code generated by mockery v2.1.0. DO NOT EDIT.

package mocks

import (
	context "context"
	io "io"

	mock "github.com/stretchr/testify/mock"

	model "github.com/mendersoftware/deployments/model"

	storage "github.com/mendersoftware/deployments/storage"

	time "time"
)

// ObjectStorage is an autogenerated mock type for the ObjectStorage type
type ObjectStorage struct {
	mock.Mock
}

// DeleteObject provides a mock function with given fields: ctx, objectId
func (_m *ObjectStorage) DeleteObject(ctx context.Context, objectId string) error {
	ret := _m.Called(ctx, objectId)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, objectId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteRequest provides a mock function with given fields: ctx, objectId, duration
func (_m *ObjectStorage) DeleteRequest(ctx context.Context, objectId string, duration time.Duration) (*model.Link, error) {
	ret := _m.Called(ctx, objectId, duration)

	var r0 *model.Link
	if rf, ok := ret.Get(0).(func(context.Context, string, time.Duration) *model.Link); ok {
		r0 = rf(ctx, objectId, duration)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Link)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, time.Duration) error); ok {
		r1 = rf(ctx, objectId, duration)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRequest provides a mock function with given fields: ctx, objectId, duration, fileName
func (_m *ObjectStorage) GetRequest(ctx context.Context, objectId string, duration time.Duration, fileName string) (*model.Link, error) {
	ret := _m.Called(ctx, objectId, duration, fileName)

	var r0 *model.Link
	if rf, ok := ret.Get(0).(func(context.Context, string, time.Duration, string) *model.Link); ok {
		r0 = rf(ctx, objectId, duration, fileName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Link)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, time.Duration, string) error); ok {
		r1 = rf(ctx, objectId, duration, fileName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HealthCheck provides a mock function with given fields: ctx
func (_m *ObjectStorage) HealthCheck(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PutObject provides a mock function with given fields: ctx, objectId, src
func (_m *ObjectStorage) PutObject(ctx context.Context, objectId string, src io.Reader) error {
	ret := _m.Called(ctx, objectId, src)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, io.Reader) error); ok {
		r0 = rf(ctx, objectId, src)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PutRequest provides a mock function with given fields: ctx, objectId, duration
func (_m *ObjectStorage) PutRequest(ctx context.Context, objectId string, duration time.Duration) (*model.Link, error) {
	ret := _m.Called(ctx, objectId, duration)

	var r0 *model.Link
	if rf, ok := ret.Get(0).(func(context.Context, string, time.Duration) *model.Link); ok {
		r0 = rf(ctx, objectId, duration)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Link)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, time.Duration) error); ok {
		r1 = rf(ctx, objectId, duration)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StatObject provides a mock function with given fields: ctx, objectId
func (_m *ObjectStorage) StatObject(ctx context.Context, objectId string) (*storage.ObjectInfo, error) {
	ret := _m.Called(ctx, objectId)

	var r0 *storage.ObjectInfo
	if rf, ok := ret.Get(0).(func(context.Context, string) *storage.ObjectInfo); ok {
		r0 = rf(ctx, objectId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*storage.ObjectInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, objectId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
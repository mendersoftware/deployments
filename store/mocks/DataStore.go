// Copyright 2019 Northern.tech AS
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
package mocks

import context "context"
import mock "github.com/stretchr/testify/mock"
import model "github.com/mendersoftware/deployments/model"

import time "time"

// DataStore is an auto-generated mock type for the DataStore type
type DataStore struct {
	mock.Mock
}

// AbortDeviceDeployments provides a mock function with given fields: ctx, deploymentID
func (_m *DataStore) AbortDeviceDeployments(ctx context.Context, deploymentID string) error {
	ret := _m.Called(ctx, deploymentID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, deploymentID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AggregateDeviceDeploymentByStatus provides a mock function with given fields: ctx, id
func (_m *DataStore) AggregateDeviceDeploymentByStatus(ctx context.Context, id string) (model.Stats, error) {
	ret := _m.Called(ctx, id)

	var r0 model.Stats
	if rf, ok := ret.Get(0).(func(context.Context, string) model.Stats); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(model.Stats)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// AssignArtifact provides a mock function with given fields: ctx, deviceID, deploymentID, artifact
func (_m *DataStore) AssignArtifact(ctx context.Context, deviceID string, deploymentID string, artifact *model.SoftwareImage) error {
	ret := _m.Called(ctx, deviceID, deploymentID, artifact)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *model.SoftwareImage) error); ok {
		r0 = rf(ctx, deviceID, deploymentID, artifact)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DecommissionDeviceDeployments provides a mock function with given fields: ctx, deviceId
func (_m *DataStore) DecommissionDeviceDeployments(ctx context.Context, deviceId string) error {
	ret := _m.Called(ctx, deviceId)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, deviceId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteDeployment provides a mock function with given fields: ctx, id
func (_m *DataStore) DeleteDeployment(ctx context.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteImage provides a mock function with given fields: ctx, id
func (_m *DataStore) DeleteImage(ctx context.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeviceCountByDeployment provides a mock function with given fields: ctx, id
func (_m *DataStore) DeviceCountByDeployment(ctx context.Context, id string) (int, error) {
	ret := _m.Called(ctx, id)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, string) int); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExistAssignedImageWithIDAndStatuses provides a mock function with given fields: ctx, id, statuses
func (_m *DataStore) ExistAssignedImageWithIDAndStatuses(ctx context.Context, id string, statuses ...string) (bool, error) {
	ret := _m.Called(ctx, id, statuses)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string, ...string) bool); ok {
		r0 = rf(ctx, id, statuses...)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, ...string) error); ok {
		r1 = rf(ctx, id, statuses...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExistByArtifactId provides a mock function with given fields: ctx, id
func (_m *DataStore) ExistByArtifactId(ctx context.Context, id string) (bool, error) {
	ret := _m.Called(ctx, id)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExistUnfinishedByArtifactId provides a mock function with given fields: ctx, id
func (_m *DataStore) ExistUnfinishedByArtifactId(ctx context.Context, id string) (bool, error) {
	ret := _m.Called(ctx, id)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Exists provides a mock function with given fields: ctx, id
func (_m *DataStore) Exists(ctx context.Context, id string) (bool, error) {
	ret := _m.Called(ctx, id)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Find provides a mock function with given fields: ctx, query
func (_m *DataStore) Find(ctx context.Context, query model.Query) ([]*model.Deployment, error) {
	ret := _m.Called(ctx, query)

	var r0 []*model.Deployment
	if rf, ok := ret.Get(0).(func(context.Context, model.Query) []*model.Deployment); ok {
		r0 = rf(ctx, query)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.Deployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, model.Query) error); ok {
		r1 = rf(ctx, query)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindAll provides a mock function with given fields: ctx
func (_m *DataStore) FindAll(ctx context.Context) ([]*model.SoftwareImage, error) {
	ret := _m.Called(ctx)

	var r0 []*model.SoftwareImage
	if rf, ok := ret.Get(0).(func(context.Context) []*model.SoftwareImage); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.SoftwareImage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindAllDeploymentsForDeviceIDWithStatuses provides a mock function with given fields: ctx, deviceID, statuses
func (_m *DataStore) FindAllDeploymentsForDeviceIDWithStatuses(ctx context.Context, deviceID string, statuses ...string) ([]model.DeviceDeployment, error) {
	ret := _m.Called(ctx, deviceID, statuses)

	var r0 []model.DeviceDeployment
	if rf, ok := ret.Get(0).(func(context.Context, string, ...string) []model.DeviceDeployment); ok {
		r0 = rf(ctx, deviceID, statuses...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.DeviceDeployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, ...string) error); ok {
		r1 = rf(ctx, deviceID, statuses...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindDeploymentByID provides a mock function with given fields: ctx, id
func (_m *DataStore) FindDeploymentByID(ctx context.Context, id string) (*model.Deployment, error) {
	ret := _m.Called(ctx, id)

	var r0 *model.Deployment
	if rf, ok := ret.Get(0).(func(context.Context, string) *model.Deployment); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Deployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindImageByID provides a mock function with given fields: ctx, id
func (_m *DataStore) FindImageByID(ctx context.Context, id string) (*model.SoftwareImage, error) {
	ret := _m.Called(ctx, id)

	var r0 *model.SoftwareImage
	if rf, ok := ret.Get(0).(func(context.Context, string) *model.SoftwareImage); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.SoftwareImage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindOldestDeploymentForDeviceIDWithStatuses provides a mock function with given fields: ctx, deviceID, statuses
func (_m *DataStore) FindOldestDeploymentForDeviceIDWithStatuses(ctx context.Context, deviceID string, statuses ...string) (*model.DeviceDeployment, error) {
	ret := _m.Called(ctx, deviceID, statuses)

	var r0 *model.DeviceDeployment
	if rf, ok := ret.Get(0).(func(context.Context, string, ...string) *model.DeviceDeployment); ok {
		r0 = rf(ctx, deviceID, statuses...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.DeviceDeployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, ...string) error); ok {
		r1 = rf(ctx, deviceID, statuses...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindUnfinishedByID provides a mock function with given fields: ctx, id
func (_m *DataStore) FindUnfinishedByID(ctx context.Context, id string) (*model.Deployment, error) {
	ret := _m.Called(ctx, id)

	var r0 *model.Deployment
	if rf, ok := ret.Get(0).(func(context.Context, string) *model.Deployment); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Deployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Finish provides a mock function with given fields: ctx, id, when
func (_m *DataStore) Finish(ctx context.Context, id string, when time.Time) error {
	ret := _m.Called(ctx, id, when)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, time.Time) error); ok {
		r0 = rf(ctx, id, when)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetDeviceDeploymentLog provides a mock function with given fields: ctx, deviceID, deploymentID
func (_m *DataStore) GetDeviceDeploymentLog(ctx context.Context, deviceID string, deploymentID string) (*model.DeploymentLog, error) {
	ret := _m.Called(ctx, deviceID, deploymentID)

	var r0 *model.DeploymentLog
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *model.DeploymentLog); ok {
		r0 = rf(ctx, deviceID, deploymentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.DeploymentLog)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, deviceID, deploymentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDeviceDeploymentStatus provides a mock function with given fields: ctx, deploymentID, deviceID
func (_m *DataStore) GetDeviceDeploymentStatus(ctx context.Context, deploymentID string, deviceID string) (string, error) {
	ret := _m.Called(ctx, deploymentID, deviceID)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, string, string) string); ok {
		r0 = rf(ctx, deploymentID, deviceID)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, deploymentID, deviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDeviceStatusesForDeployment provides a mock function with given fields: ctx, deploymentID
func (_m *DataStore) GetDeviceStatusesForDeployment(ctx context.Context, deploymentID string) ([]model.DeviceDeployment, error) {
	ret := _m.Called(ctx, deploymentID)

	var r0 []model.DeviceDeployment
	if rf, ok := ret.Get(0).(func(context.Context, string) []model.DeviceDeployment); ok {
		r0 = rf(ctx, deploymentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.DeviceDeployment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, deploymentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLimit provides a mock function with given fields: ctx, name
func (_m *DataStore) GetLimit(ctx context.Context, name string) (*model.Limit, error) {
	ret := _m.Called(ctx, name)

	var r0 *model.Limit
	if rf, ok := ret.Get(0).(func(context.Context, string) *model.Limit); ok {
		r0 = rf(ctx, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Limit)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetReleases provides a mock function with given fields: ctx, filt
func (_m *DataStore) GetReleases(ctx context.Context, filt *model.ReleaseFilter) ([]model.Release, error) {
	ret := _m.Called(ctx, filt)

	var r0 []model.Release
	if rf, ok := ret.Get(0).(func(context.Context, *model.ReleaseFilter) []model.Release); ok {
		r0 = rf(ctx, filt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.Release)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *model.ReleaseFilter) error); ok {
		r1 = rf(ctx, filt)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HasDeploymentForDevice provides a mock function with given fields: ctx, deploymentID, deviceID
func (_m *DataStore) HasDeploymentForDevice(ctx context.Context, deploymentID string, deviceID string) (bool, error) {
	ret := _m.Called(ctx, deploymentID, deviceID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string, string) bool); ok {
		r0 = rf(ctx, deploymentID, deviceID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, deploymentID, deviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ImageByIdsAndDeviceType provides a mock function with given fields: ctx, ids, deviceType
func (_m *DataStore) ImageByIdsAndDeviceType(ctx context.Context, ids []string, deviceType string) (*model.SoftwareImage, error) {
	ret := _m.Called(ctx, ids, deviceType)

	var r0 *model.SoftwareImage
	if rf, ok := ret.Get(0).(func(context.Context, []string, string) *model.SoftwareImage); ok {
		r0 = rf(ctx, ids, deviceType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.SoftwareImage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, []string, string) error); ok {
		r1 = rf(ctx, ids, deviceType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ImageByNameAndDeviceType provides a mock function with given fields: ctx, name, deviceType
func (_m *DataStore) ImageByNameAndDeviceType(ctx context.Context, name string, deviceType string) (*model.SoftwareImage, error) {
	ret := _m.Called(ctx, name, deviceType)

	var r0 *model.SoftwareImage
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *model.SoftwareImage); ok {
		r0 = rf(ctx, name, deviceType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.SoftwareImage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, name, deviceType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ImagesByName provides a mock function with given fields: ctx, artifactName
func (_m *DataStore) ImagesByName(ctx context.Context, artifactName string) ([]*model.SoftwareImage, error) {
	ret := _m.Called(ctx, artifactName)

	var r0 []*model.SoftwareImage
	if rf, ok := ret.Get(0).(func(context.Context, string) []*model.SoftwareImage); ok {
		r0 = rf(ctx, artifactName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.SoftwareImage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, artifactName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// InsertDeployment provides a mock function with given fields: ctx, deployment
func (_m *DataStore) InsertDeployment(ctx context.Context, deployment *model.Deployment) error {
	ret := _m.Called(ctx, deployment)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.Deployment) error); ok {
		r0 = rf(ctx, deployment)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// InsertImage provides a mock function with given fields: ctx, image
func (_m *DataStore) InsertImage(ctx context.Context, image *model.SoftwareImage) error {
	ret := _m.Called(ctx, image)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.SoftwareImage) error); ok {
		r0 = rf(ctx, image)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// InsertMany provides a mock function with given fields: ctx, deployment
func (_m *DataStore) InsertMany(ctx context.Context, deployment ...*model.DeviceDeployment) error {
	ret := _m.Called(ctx, deployment)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, ...*model.DeviceDeployment) error); ok {
		r0 = rf(ctx, deployment...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IsArtifactUnique provides a mock function with given fields: ctx, artifactName, deviceTypesCompatible
func (_m *DataStore) IsArtifactUnique(ctx context.Context, artifactName string, deviceTypesCompatible []string) (bool, error) {
	ret := _m.Called(ctx, artifactName, deviceTypesCompatible)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string, []string) bool); ok {
		r0 = rf(ctx, artifactName, deviceTypesCompatible)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, []string) error); ok {
		r1 = rf(ctx, artifactName, deviceTypesCompatible)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ProvisionTenant provides a mock function with given fields: ctx, tenantId
func (_m *DataStore) ProvisionTenant(ctx context.Context, tenantId string) error {
	ret := _m.Called(ctx, tenantId)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, tenantId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SaveDeviceDeploymentLog provides a mock function with given fields: ctx, log
func (_m *DataStore) SaveDeviceDeploymentLog(ctx context.Context, log model.DeploymentLog) error {
	ret := _m.Called(ctx, log)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, model.DeploymentLog) error); ok {
		r0 = rf(ctx, log)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Update provides a mock function with given fields: ctx, image
func (_m *DataStore) Update(ctx context.Context, image *model.SoftwareImage) (bool, error) {
	ret := _m.Called(ctx, image)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, *model.SoftwareImage) bool); ok {
		r0 = rf(ctx, image)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *model.SoftwareImage) error); ok {
		r1 = rf(ctx, image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateDeviceDeploymentLogAvailability provides a mock function with given fields: ctx, deviceID, deploymentID, log
func (_m *DataStore) UpdateDeviceDeploymentLogAvailability(ctx context.Context, deviceID string, deploymentID string, log bool) error {
	ret := _m.Called(ctx, deviceID, deploymentID, log)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, bool) error); ok {
		r0 = rf(ctx, deviceID, deploymentID, log)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateDeviceDeploymentStatus provides a mock function with given fields: ctx, deviceID, deploymentID, status
func (_m *DataStore) UpdateDeviceDeploymentStatus(ctx context.Context, deviceID string, deploymentID string, status model.DeviceDeploymentStatus) (string, error) {
	ret := _m.Called(ctx, deviceID, deploymentID, status)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, string, string, model.DeviceDeploymentStatus) string); ok {
		r0 = rf(ctx, deviceID, deploymentID, status)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, model.DeviceDeploymentStatus) error); ok {
		r1 = rf(ctx, deviceID, deploymentID, status)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateStats provides a mock function with given fields: ctx, id, state_from, state_to
func (_m *DataStore) UpdateStats(ctx context.Context, id string, state_from string, state_to string) error {
	ret := _m.Called(ctx, id, state_from, state_to)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) error); ok {
		r0 = rf(ctx, id, state_from, state_to)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateStatsAndFinishDeployment provides a mock function with given fields: ctx, id, stats
func (_m *DataStore) UpdateStatsAndFinishDeployment(ctx context.Context, id string, stats model.Stats) error {
	ret := _m.Called(ctx, id, stats)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, model.Stats) error); ok {
		r0 = rf(ctx, id, stats)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

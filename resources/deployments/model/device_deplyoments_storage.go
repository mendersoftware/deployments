// Copyright 2016 Mender Software AS
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

package model

import (
	"time"

	"github.com/mendersoftware/deployments/resources/deployments"
)

// Device deployment storage
type DeviceDeploymentStorage interface {
	InsertMany(deployment ...*deployments.DeviceDeployment) error
	ExistAssignedImageWithIDAndStatuses(id string, statuses ...string) (bool, error)
	FindOldestDeploymentForDeviceIDWithStatuses(deviceID string, statuses ...string) (*deployments.DeviceDeployment, error)
	UpdateDeviceDeploymentStatus(deviceID string, deploymentID string, status string, finishTime *time.Time) (string, error)
	UpdateDeviceDeploymentLogAvailability(deviceID string, deploymentID string, log bool) error
	AggregateDeviceDeploymentByStatus(id string) (deployments.Stats, error)
	GetDeviceStatusesForDeployment(deploymentID string) ([]deployments.DeviceDeployment, error)
	HasDeploymentForDevice(deploymentID string, deviceID string) (bool, error)
	GetDeviceDeploymentStatus(deploymentID string, deviceID string) (string, error)
	AbortDeviceDeployments(deploymentID string) error
}

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

package model

import (
	"context"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/resources/deployments"
)

// Device deployment storage
type DeviceDeploymentStorage interface {
	InsertMany(ctx context.Context,
		deployment ...*deployments.DeviceDeployment) error
	ExistAssignedImageWithIDAndStatuses(ctx context.Context,
		id string, statuses ...string) (bool, error)
	FindOldestDeploymentForDeviceIDWithStatuses(ctx context.Context,
		deviceID string, statuses ...string) (*deployments.DeviceDeployment, error)
	FindAllDeploymentsForDeviceIDWithStatuses(ctx context.Context,
		deviceID string, statuses ...string) ([]deployments.DeviceDeployment, error)

	UpdateDeviceDeploymentStatus(ctx context.Context, deviceID string,
		deploymentID string, status deployments.DeviceDeploymentStatus) (string, error)

	UpdateDeviceDeploymentLogAvailability(ctx context.Context,
		deviceID string, deploymentID string, log bool) error
	AssignArtifact(ctx context.Context, deviceID string,
		deploymentID string, artifact *model.SoftwareImage) error
	AggregateDeviceDeploymentByStatus(ctx context.Context,
		id string) (deployments.Stats, error)
	GetDeviceStatusesForDeployment(ctx context.Context,
		deploymentID string) ([]deployments.DeviceDeployment, error)
	HasDeploymentForDevice(ctx context.Context,
		deploymentID string, deviceID string) (bool, error)
	GetDeviceDeploymentStatus(ctx context.Context,
		deploymentID string, deviceID string) (string, error)
	AbortDeviceDeployments(ctx context.Context, deploymentID string) error
	DecommissionDeviceDeployments(ctx context.Context, deviceId string) error
}

// Copyright 2017 Northern.tech AS
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

package controller

import (
	"context"
	"errors"

	"github.com/mendersoftware/deployments/resources/deployments"
)

// Errors
var (
	ErrModelMissingInput       = errors.New("Missing input deployment data")
	ErrModelInvalidDeviceID    = errors.New("Invalid device ID")
	ErrModelDeploymentNotFound = errors.New("Deployment not found")
	ErrModelInternal           = errors.New("Internal error")
	ErrStorageInvalidLog       = errors.New("Invalid deployment log")
	ErrStorageNotFound         = errors.New("Not found")
	ErrDeploymentAborted       = errors.New("Deployment aborted")
	ErrDeviceDecommissioned    = errors.New("Device decommissioned")
)

// Domain model for deployment
type DeploymentsModel interface {
	CreateDeployment(ctx context.Context,
		constructor *deployments.DeploymentConstructor) (string, error)
	GetDeployment(ctx context.Context, deploymentID string) (*deployments.Deployment, error)
	IsDeploymentFinished(ctx context.Context, deploymentID string) (bool, error)
	AbortDeployment(ctx context.Context, deploymentID string) error
	GetDeploymentStats(ctx context.Context, deploymentID string) (deployments.Stats, error)
	GetDeploymentForDeviceWithCurrent(ctx context.Context, deviceID string,
		current deployments.InstalledDeviceDeployment) (*deployments.DeploymentInstructions, error)
	HasDeploymentForDevice(ctx context.Context, deploymentID string,
		deviceID string) (bool, error)
	UpdateDeviceDeploymentStatus(ctx context.Context, deploymentID string,
		deviceID string, status string) error
	GetDeviceStatusesForDeployment(ctx context.Context,
		deploymentID string) ([]deployments.DeviceDeployment, error)
	LookupDeployment(ctx context.Context,
		query deployments.Query) ([]*deployments.Deployment, error)
	SaveDeviceDeploymentLog(ctx context.Context, deviceID string,
		deploymentID string, logs []deployments.LogMessage) error
	GetDeviceDeploymentLog(ctx context.Context,
		deviceID, deploymentID string) (*deployments.DeploymentLog, error)
	DecommissionDevice(ctx context.Context, deviceID string) error
}

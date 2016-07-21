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

package controller

import (
	"errors"
	"github.com/mendersoftware/deployments/resources/deployments"
)

// Errors
var (
	ErrModelMissingInput       = errors.New("Missing input deplyoment data")
	ErrModelInvalidDeviceID    = errors.New("Invalid device ID")
	ErrModelDeploymentNotFound = errors.New("Deployment not found")
	ErrModelInternal           = errors.New("Internal error")
)

// Domain model for deployment
type DeploymentsModel interface {
	CreateDeployment(constructor *deployments.DeploymentConstructor) (string, error)
	GetDeployment(deploymentID string) (*deployments.Deployment, error)
	GetDeploymentStats(deploymentID string) (deployments.Stats, error)
	GetDeploymentForDevice(deviceID string) (*deployments.DeploymentInstructions, error)
	UpdateDeviceDeploymentStatus(deploymentID string, deviceID string, status string) error
	GetDeviceStatusesForDeployment(deploymentID string) ([]deployments.DeviceDeployment, error)
	LookupDeployment(query deployments.Query) ([]*deployments.Deployment, error)
	SaveDeviceDeploymentLog(deviceID string, deploymentID string, log *deployments.DeploymentLog) error
}

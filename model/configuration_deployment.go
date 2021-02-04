// Copyright 2021 Northern.tech AS
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

	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pkg/errors"
)

// ConfigurationDeploymentConstructor represent input data needed for creating new Configuraiont Deployment
type ConfigurationDeploymentConstructor struct {
	// Deployment name, required
	Name string `json:"name"`

	// A string containing a configuration object.
	// The deployments service will use it to generate configuration
	// artifact for the device.
	// The artifact will be generated when the device will ask
	// for an update.
	Configuration string `json:"configuration"`
}

// Validate validates the structure.
func (c ConfigurationDeploymentConstructor) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Name, validation.Required, lengthIn1To4096),
		validation.Field(&c.Configuration, validation.Required),
	)
}

// NewConfigurationDeployment creates new configuration deployment object, sets create data by default.
func NewDeploymentWithID(ID string) (*Deployment, error) {
	now := time.Now()

	return &Deployment{
		Created: &now,
		Id:      ID,
		Stats:   NewDeviceDeploymentStats(),
		Status:  DeploymentStatusPending,
	}, nil
}

// NewConfigurationDeploymentFromConstructor creates new Deployments object based onconfiguration deployment constructor data
func NewDeploymentFromConfigurationDeploymentConstructor(constructor *ConfigurationDeploymentConstructor, deploymentID string) (*Deployment, error) {

	deployment, err := NewDeploymentWithID(deploymentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create deployment from constructor")
	}

	deployment.DeploymentConstructor = &DeploymentConstructor{
		Name: constructor.Name,
	}

	deviceCount := 0
	deployment.DeviceCount = &deviceCount

	return deployment, nil
}

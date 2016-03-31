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
package models

import (
	"errors"
	"time"

	"github.com/satori/go.uuid"
)

var (
	ErrValidationMissingField error = errors.New("missing field")
)

const (
	DeploymentStatusPending = "pending"
)

// DeploymentConstructor represent input data needed for creating new Deployment (they differ in fields)
type DeploymentConstructor struct {
	// Deployment name, required
	Name *string `json:"name,omitempty"`

	// Software name/version, required, associated with image
	Version *string `json:"version,omitempty"`

	// List of device id's targeted for deployments, required
	Devices []string `json:"devices"`
}

func NewDeploymentConstructor() *DeploymentConstructor {
	return &DeploymentConstructor{}
}

// Validate input data
// TODO: Improve validation, some parts are missing, replace with govalidate perhaps
func (c *DeploymentConstructor) Validate() error {

	if c.Name == nil || *c.Name == "" {
		return ErrValidationMissingField
	}

	if c.Version == nil || *c.Version == "" {
		return ErrValidationMissingField
	}

	if c.Devices == nil || len(c.Devices) == 0 {
		return ErrValidationMissingField
	}

	return nil
}

type Deployment struct {
	// Auto set on create, required
	Created *time.Time `json:"created"`

	// Finished deplyment time
	Finished *time.Time `json:"finished,omitempty"`

	// Enum, required
	Status *string `json:"status"`

	// Deployment name, required
	Name *string `json:"name"`

	// Software name/version, required, associated with image
	Version *string `json:"version"`

	// Deployment id, required
	Id *string `json:"id" bson:"_id"`
}

// NewDeployment creates not deployment object, sets create data by default.
func NewDeployment() *Deployment {
	now := time.Now()
	status := DeploymentStatusPending
	id := uuid.NewV4().String()
	return &Deployment{
		Created: &now,
		Status:  &status,
		Id:      &id,
	}
}

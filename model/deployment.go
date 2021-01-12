// Copyright 2020 Northern.tech AS
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
	"encoding/json"
	"github.com/pkg/errors"
	"time"

	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/satori/go.uuid"
)

// Errors
var (
	ErrInvalidDeviceID                            = errors.New("Invalid device ID")
	ErrInvalidDeploymentDefinition                = errors.New("Invalid deployments definition")
	ErrInvalidDeploymentDefinitionNoDevices       = errors.New("Invalid deployments definition: provide list of devices or set all_devices flag")
	ErrInvalidDeploymentDefinitionConflict        = errors.New("Invalid deployments definition: list of devices provided togheter with all_devices flag")
	ErrInvalidDeploymentToGroupDefinitionConflict = errors.New("The deployment for group constructor should have neither list of devices nor all_devices flag set")
)

type DeploymentStatus string

const (
	DeploymentStatusFinished   DeploymentStatus = "finished"
	DeploymentStatusInProgress DeploymentStatus = "inprogress"
	DeploymentStatusPending    DeploymentStatus = "pending"
)

func (stat DeploymentStatus) Validate() error {
	return validation.In(
		DeploymentStatusFinished,
		DeploymentStatusInProgress,
		DeploymentStatusPending,
	).Validate(stat)
}

// DeploymentConstructor represent input data needed for creating new Deployment (they differ in fields)
type DeploymentConstructor struct {
	// Deployment name, required
	Name string `json:"name,omitempty"`

	// Artifact name to be installed required, associated with image
	ArtifactName string `json:"artifact_name,omitempty"`

	// List of device id's targeted for deployments, required
	Devices []string `json:"devices,omitempty" bson:"-"`

	// When set to true deployment will be created for all currently accepted devices
	AllDevices bool `json:"all_devices,omitempty" bson:"-"`

	// When set the deployment will be created for all accepted devices from a given group
	Group string `json:"-" bson:"-"`
}

// Validate checks structure according to valid tags
// TODO: Add custom validator to check devices array content (such us UUID formatting)
func (c DeploymentConstructor) Validate() error {
	if err := validation.ValidateStruct(&c,
		validation.Field(&c.Name, validation.Required, lengthIn1To4096),
		validation.Field(&c.ArtifactName, validation.Required, lengthIn1To4096),
		validation.Field(&c.Devices, validation.Each(validation.Required)),
	); err != nil {
		return err
	}

	if len(c.Group) == 0 {
		if len(c.Devices) == 0 && !c.AllDevices {
			return ErrInvalidDeploymentDefinitionNoDevices
		}
		if len(c.Devices) > 0 && c.AllDevices {
			return ErrInvalidDeploymentDefinitionConflict
		}
	} else {
		if len(c.Devices) > 0 || c.AllDevices {
			return ErrInvalidDeploymentToGroupDefinitionConflict
		}
	}

	return nil
}

type Deployment struct {
	// User provided field set
	*DeploymentConstructor

	// Auto set on create, required
	Created *time.Time `json:"created"`

	// Finished deployment time
	Finished *time.Time `json:"finished,omitempty"`

	// Deployment id, required
	Id string `json:"id" bson:"_id"`

	// List of artifact id's targeted for deployments, optional
	Artifacts []string `json:"artifacts,omitempty" bson:"artifacts"`

	// Aggregated device status counters.
	// Initialized with the "pending" counter set to total device count for deployment.
	// Individual counter incremented/decremented according to device status updates.
	Stats Stats `json:"-"`

	// Status is the overall deployment status
	Status DeploymentStatus `json:"status" bson:"status"`

	// Number of devices being part of the deployment
	DeviceCount *int `json:"device_count" bson:"device_count"`

	// Total number of devices targeted
	MaxDevices int `json:"max_devices,omitempty" bson:"max_devices"`

	// list of devices
	DeviceList []string `json:"-" bson:"device_list"`
}

// NewDeployment creates new deployment object, sets create data by default.
func NewDeployment() (*Deployment, error) {
	now := time.Now()

	uid := uuid.NewV4()
	id := uid.String()

	return &Deployment{
		Created:               &now,
		Id:                    id,
		DeploymentConstructor: &DeploymentConstructor{},
		Stats:                 NewDeviceDeploymentStats(),
	}, nil
}

// NewDeploymentFromConstructor creates new Deployments object based on constructor data
func NewDeploymentFromConstructor(constructor *DeploymentConstructor) (*Deployment, error) {

	deployment, err := NewDeployment()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create deployment from constructor")
	}

	deployment.DeploymentConstructor = constructor
	deployment.Status = DeploymentStatusPending

	deviceCount := 0
	deployment.DeviceCount = &deviceCount

	return deployment, nil
}

// Validate checks structure validation rules
func (d Deployment) Validate() error {
	return validation.ValidateStruct(&d,
		validation.Field(&d.DeploymentConstructor, validation.NotNil),
		validation.Field(&d.Created, validation.Required),
		validation.Field(&d.Id, validation.Required, is.UUID),
		validation.Field(&d.Artifacts, validation.Each(validation.Required)),
		validation.Field(&d.DeviceList, validation.Each(validation.Required)),
	)
}

// To be able to hide devices field, from API output provide custom marshaler
func (d *Deployment) MarshalJSON() ([]byte, error) {

	//Prevents from inheriting original MarshalJSON (if would, infinite loop)
	type Alias Deployment

	slim := struct {
		*Alias
		Devices []string `json:"devices,omitempty"`
	}{
		Alias:   (*Alias)(d),
		Devices: nil,
	}

	return json.Marshal(&slim)
}

func (d *Deployment) IsNotPending() bool {
	if d.Stats[DeviceDeploymentStatusDownloading] > 0 ||
		d.Stats[DeviceDeploymentStatusInstalling] > 0 ||
		d.Stats[DeviceDeploymentStatusRebooting] > 0 ||
		d.Stats[DeviceDeploymentStatusSuccess] > 0 ||
		d.Stats[DeviceDeploymentStatusAlreadyInst] > 0 ||
		d.Stats[DeviceDeploymentStatusFailure] > 0 ||
		d.Stats[DeviceDeploymentStatusAborted] > 0 ||
		d.Stats[DeviceDeploymentStatusNoArtifact] > 0 {

		return true
	}

	return false
}

func (d *Deployment) IsFinished() bool {
	if d.MaxDevices <= 0 || ((d.Stats[DeviceDeploymentStatusAlreadyInst] +
		d.Stats[DeviceDeploymentStatusSuccess] +
		d.Stats[DeviceDeploymentStatusFailure] +
		d.Stats[DeviceDeploymentStatusNoArtifact] +
		d.Stats[DeviceDeploymentStatusDecommissioned] +
		d.Stats[DeviceDeploymentStatusAborted]) >= d.MaxDevices) {
		return true
	}

	return false
}

func (d *Deployment) GetStatus() DeploymentStatus {
	if d.IsFinished() {
		return DeploymentStatusFinished
	} else if d.IsNotPending() {
		return DeploymentStatusInProgress
	} else {
		return DeploymentStatusPending
	}
}

type StatusQuery int

const (
	StatusQueryAny StatusQuery = iota
	StatusQueryPending
	StatusQueryInProgress
	StatusQueryFinished
	StatusQueryAborted
)

// Deployment lookup query
type Query struct {
	// match deployments by text by looking at deployment name and artifact name
	SearchText string

	// deployment status
	Status StatusQuery
	Limit  int
	Skip   int
	// only return deployments between timestamp range
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

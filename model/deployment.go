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

	"github.com/asaskevich/govalidator"
	"github.com/satori/go.uuid"
)

// Errors
var (
	ErrInvalidDeviceID             = errors.New("Invalid device ID")
	ErrInvalidDeploymentDefinition = errors.New("Invalid deployments definition")
)

const (
	DeploymentStatusFinished   = "finished"
	DeploymentStatusInProgress = "inprogress"
	DeploymentStatusPending    = "pending"
)

// DeploymentConstructor represent input data needed for creating new Deployment (they differ in fields)
type DeploymentConstructor struct {
	// Deployment name, required
	Name *string `json:"name,omitempty" valid:"length(1|4096),required"`

	// Artifact name to be installed required, associated with image
	ArtifactName *string `json:"artifact_name,omitempty" valid:"length(1|4096),required"`

	// List of device id's targeted for deployments, required
	Devices []string `json:"devices,omitempty" valid:"required" bson:"-"`
}

// Validate checks structure according to valid tags
// TODO: Add custom validator to check devices array content (such us UUID formatting)
func (c *DeploymentConstructor) Validate(groupDeployment string) error {
	if len(groupDeployment) > 0 {
		if c.Name == nil || len(*c.Name) < 1 {
			return ErrInvalidDeploymentDefinition
		}
		return nil
	}

	if _, err := govalidator.ValidateStruct(c); err != nil {
		return err
	}

	for _, id := range c.Devices {
		if govalidator.IsNull(id) {
			return ErrInvalidDeviceID
		}
	}

	return nil
}

type Deployment struct {
	// User provided field set
	*DeploymentConstructor `valid:"required"`

	// Auto set on create, required
	Created *time.Time `json:"created" valid:"required"`

	// Finished deployment time
	Finished *time.Time `json:"finished,omitempty" valid:"optional"`

	// Deployment id, required
	Id *string `json:"id" bson:"_id" valid:"uuidv4,required"`

	// List of artifact id's targeted for deployments, optional
	Artifacts []string `json:"artifacts,omitempty" bson:"artifacts"`

	// Aggregated device status counters.
	// Initialized with the "pending" counter set to total device count for deployment.
	// Individual counter incremented/decremented according to device status updates.
	Stats Stats `json:"-"`

	// Status is the overall deployment status
	Status string `json:"status" bson:"status"`

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

	uid, err := uuid.NewV4()
	if err != nil {
		return nil, errors.New("failed to generate uuid")
	}

	id := uid.String()

	return &Deployment{
		Created:               &now,
		Id:                    &id,
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

// Validate checks structure according to valid tags
func (d *Deployment) Validate() error {
	_, err := govalidator.ValidateStruct(d)
	return err
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

func (d *Deployment) GetStatus() string {
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

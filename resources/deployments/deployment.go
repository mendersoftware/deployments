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

package deployments

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/satori/go.uuid"
)

// Errors
var (
	ErrInvalidDeviceID = errors.New("Invalid device ID")
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

func NewDeploymentConstructor() *DeploymentConstructor {
	return &DeploymentConstructor{}
}

// Validate checkes structure according to valid tags
// TODO: Add custom validator to check devices array content (such us UUID formatting)
func (c *DeploymentConstructor) Validate() error {
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

	// Finished deplyment time
	Finished *time.Time `json:"finished,omitempty" valid:"optional"`

	// Deployment id, required
	Id *string `json:"id" bson:"_id" valid:"uuidv4,required"`

	// Aggregated device status counters.
	// Initialized with the "pending" counter set to total device count for deployment.
	// Individual counter incremented/decremented according to device status updates.
	Stats map[string]int `json:"stats"`
}

// NewDeployment creates new deployment object, sets create data by default.
func NewDeployment() *Deployment {
	now := time.Now()
	id := uuid.NewV4().String()

	return &Deployment{
		Created: &now,
		Id:      &id,
		DeploymentConstructor: NewDeploymentConstructor(),
		Stats: NewDeviceDeploymentStats(),
	}
}

// NewDeploymentFromConstructor creates new Deployments object based on constructor data
func NewDeploymentFromConstructor(constructor *DeploymentConstructor) *Deployment {

	deployment := NewDeployment()
	deployment.DeploymentConstructor = constructor
	if constructor != nil && constructor.Devices != nil {
		deployment.Stats[DeviceDeploymentStatusPending] = len(constructor.Devices)
	}
	return deployment
}

// Validate checkes structure according to valid tags
func (d *Deployment) Validate() error {
	_, err := govalidator.ValidateStruct(d)
	return err
}

// To be able to hide devices field, from API output provice custom marshaler
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

func (d *Deployment) IsInProgress() bool {
	active := []string{
		DeviceDeploymentStatusRebooting,
		DeviceDeploymentStatusInstalling,
		DeviceDeploymentStatusDownloading,
	}

	var acount int
	for _, s := range active {
		acount += d.Stats[s]
	}

	if acount != 0 {
		return true
	}
	return false
}

func (d *Deployment) IsFinished() bool {
	// check if there are downloading/rebooting/installing devices
	if d.IsInProgress() {
		return false
	}

	// check if there are pending devices
	if d.Stats[DeviceDeploymentStatusPending] != 0 {
		return false
	}

	return true
}

func (d *Deployment) IsPending() bool {
	if d.IsInProgress() {
		return false
	}

	if d.IsFinished() {
		return false
	}

	return true
}

type StatusQuery int

const (
	StatusQueryAny StatusQuery = iota
	StatusQueryPending
	StatusQueryInProgress
	StatusQueryFinished
)

// Deployment lookup query
type Query struct {
	// match deployments by text by looking at deployment name and artifact name
	SearchText string
	// deployment status
	Status StatusQuery
}

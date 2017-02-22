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
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/deployments/resources/images"
	"github.com/satori/go.uuid"
)

// Deployment statuses
const (
	DeviceDeploymentStatusDownloading = "downloading"
	DeviceDeploymentStatusInstalling  = "installing"
	DeviceDeploymentStatusRebooting   = "rebooting"
	DeviceDeploymentStatusPending     = "pending"
	DeviceDeploymentStatusSuccess     = "success"
	DeviceDeploymentStatusFailure     = "failure"
	DeviceDeploymentStatusNoArtifact  = "noartifact"
	DeviceDeploymentStatusAlreadyInst = "already-installed"
	DeviceDeploymentStatusAborted     = "aborted"
)

type DeviceDeployment struct {
	// Internal field of initial creation of deployment
	Created *time.Time `json:"created" valid:"required"`

	// Update finish time
	Finished *time.Time `json:"finished,omitempty" valid:"-"`

	// Status
	Status *string `json:"status" valid:"required"`

	// Device id
	DeviceId *string `json:"id" valid:"required"`

	// Deployment id
	DeploymentId *string `json:"-" valid:"uuidv4,required"`

	// ID
	Id *string `json:"-" bson:"_id" valid:"uuidv4,required"`

	// Assigned software image
	Image *images.SoftwareImage `json:"-" valid:"-"`

	// Target device type
	DeviceType *string `json:"device_type,omitempty" valid:"-"`

	// Presence of deployment log
	IsLogAvailable bool `json:"log" valid:"-" bson:"log"`
}

func NewDeviceDeployment(deviceId, deploymentId string) *DeviceDeployment {

	now := time.Now()
	initStatus := DeviceDeploymentStatusPending
	id := uuid.NewV4().String()

	return &DeviceDeployment{
		Status:         &initStatus,
		DeviceId:       &deviceId,
		DeploymentId:   &deploymentId,
		Id:             &id,
		Created:        &now,
		IsLogAvailable: false,
	}
}

func (d *DeviceDeployment) Validate() error {
	_, err := govalidator.ValidateStruct(d)
	return err
}

// Deployment statistics wrapper, each value carries a count of deployments
// aggregated by state.
type Stats map[string]int

func NewDeviceDeploymentStats() Stats {
	statuses := []string{
		DeviceDeploymentStatusNoArtifact,
		DeviceDeploymentStatusFailure,
		DeviceDeploymentStatusSuccess,
		DeviceDeploymentStatusPending,
		DeviceDeploymentStatusRebooting,
		DeviceDeploymentStatusInstalling,
		DeviceDeploymentStatusDownloading,
		DeviceDeploymentStatusAlreadyInst,
		DeviceDeploymentStatusAborted,
	}

	s := make(Stats)

	// populate statuses with 0s
	for _, v := range statuses {
		s[v] = 0
	}

	return s
}

func IsDeviceDeploymentStatusFinished(status string) bool {
	if status == DeviceDeploymentStatusFailure || status == DeviceDeploymentStatusSuccess ||
		status == DeviceDeploymentStatusNoArtifact || status == DeviceDeploymentStatusAlreadyInst ||
		status == DeviceDeploymentStatusAborted {
		return true
	}
	return false
}

// ActiveDeploymentStatuses lists statuses that represent deployment in active state (not finished).
func ActiveDeploymentStatuses() []string {
	return []string{
		DeviceDeploymentStatusPending,
		DeviceDeploymentStatusDownloading,
		DeviceDeploymentStatusInstalling,
		DeviceDeploymentStatusRebooting,
	}
}

// InstalledDeviceDeployment describes a deployment currently installed on the
// device, usually reported by a device
type InstalledDeviceDeployment struct {
	Artifact   string `valid:"required"`
	DeviceType string `valid:"required"`
}

func (i *InstalledDeviceDeployment) Validate() error {
	_, err := govalidator.ValidateStruct(i)
	return err
}

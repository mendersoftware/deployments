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
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

// Deployment statuses
const (
	DeviceDeploymentStatusDownloading    = "downloading"
	DeviceDeploymentStatusInstalling     = "installing"
	DeviceDeploymentStatusRebooting      = "rebooting"
	DeviceDeploymentStatusPending        = "pending"
	DeviceDeploymentStatusSuccess        = "success"
	DeviceDeploymentStatusFailure        = "failure"
	DeviceDeploymentStatusNoArtifact     = "noartifact"
	DeviceDeploymentStatusAlreadyInst    = "already-installed"
	DeviceDeploymentStatusAborted        = "aborted"
	DeviceDeploymentStatusDecommissioned = "decommissioned"
)

// DeviceDeploymentStatus is a helper type for reporting status changes through
// the layers
type DeviceDeploymentStatus struct {
	// status reported by device
	Status string `valid:"required"`
	// substate reported by device
	SubState *string
	// finish time
	FinishTime *time.Time
}

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
	Image *Image `json:"-" valid:"-"`

	// Target device type
	DeviceType *string `json:"device_type,omitempty" valid:"-"`

	// Presence of deployment log
	IsLogAvailable bool `json:"log" valid:"-" bson:"log"`

	// Device reported substate
	SubState *string `json:"substate,omitempty" valid:"-" bson:"substate"`
}

func NewDeviceDeployment(deviceId, deploymentId string) (*DeviceDeployment, error) {

	now := time.Now()
	initStatus := DeviceDeploymentStatusPending

	uid, err := uuid.NewV4()
	if err != nil {
		return nil, errors.New("failed to generate uuid")
	}

	id := uid.String()

	return &DeviceDeployment{
		Status:         &initStatus,
		DeviceId:       &deviceId,
		DeploymentId:   &deploymentId,
		Id:             &id,
		Created:        &now,
		IsLogAvailable: false,
	}, nil
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
		DeviceDeploymentStatusDecommissioned,
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
		status == DeviceDeploymentStatusAborted || status == DeviceDeploymentStatusDecommissioned {
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

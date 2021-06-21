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

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type DeviceDeploymentStatus string

// Deployment statuses
const (
	DeviceDeploymentStatusDownloading        DeviceDeploymentStatus = "downloading"
	DeviceDeploymentStatusInstalling         DeviceDeploymentStatus = "installing"
	DeviceDeploymentStatusRebooting          DeviceDeploymentStatus = "rebooting"
	DeviceDeploymentStatusPending            DeviceDeploymentStatus = "pending"
	DeviceDeploymentStatusSuccess            DeviceDeploymentStatus = "success"
	DeviceDeploymentStatusFailure            DeviceDeploymentStatus = "failure"
	DeviceDeploymentStatusNoArtifact         DeviceDeploymentStatus = "noartifact"
	DeviceDeploymentStatusAlreadyInst        DeviceDeploymentStatus = "already-installed"
	DeviceDeploymentStatusAborted            DeviceDeploymentStatus = "aborted"
	DeviceDeploymentStatusDecommissioned     DeviceDeploymentStatus = "decommissioned"
	DeviceDeploymentStatusPauseBeforeInstall DeviceDeploymentStatus = "pause_before_installing"
	DeviceDeploymentStatusPauseBeforeCommit  DeviceDeploymentStatus = "pause_before_committing"
	DeviceDeploymentStatusPauseBeforeReboot  DeviceDeploymentStatus = "pause_before_rebooting"
)

var allStatuses = []interface{}{ // NOTE: []DeviceDeploymentStatus
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
	DeviceDeploymentStatusPauseBeforeInstall,
	DeviceDeploymentStatusPauseBeforeCommit,
	DeviceDeploymentStatusPauseBeforeReboot,
}

func (stat DeviceDeploymentStatus) Validate() error {
	return validation.In(allStatuses...).
		Validate(stat)
}

// DeviceDeploymentStatus is a helper type for reporting status changes through
// the layers
type DeviceDeploymentState struct {
	// status reported by device
	Status DeviceDeploymentStatus
	// substate reported by device
	SubState string `json:",omitempty" bson:",omitempty"`
	// finish time
	FinishTime *time.Time `json:",omitempty" bson:",omitempty"`
}

func (state DeviceDeploymentState) Validate() error {
	return validation.ValidateStruct(&state,
		validation.Field(&state.Status, validation.Required),
	)
}

type DeviceDeployment struct {
	// Internal field of initial creation of deployment
	Created *time.Time `json:"created" bson:"created"`

	// Update finish time
	Finished *time.Time `json:"finished,omitempty" bson:"finished,omitempty"`

	// Status
	Status DeviceDeploymentStatus `json:"status" bson:"status"`

	// Device id
	DeviceId string `json:"id" bson:"deviceid"`

	// Deployment id
	DeploymentId string `json:"-" bson:"deploymentid"`

	// ID
	Id string `json:"-" bson:"_id"`

	// Assigned software image
	Image *Image `json:"-"`

	// Target device type
	DeviceType string `json:"device_type,omitempty" bson:"devicetype"`

	// Presence of deployment log
	IsLogAvailable bool `json:"log" bson:"log"`

	// Device reported substate
	SubState string `json:"substate,omitempty" bson:"substate,omitempty"`
}

func NewDeviceDeployment(deviceId, deploymentId string) *DeviceDeployment {

	now := time.Now()

	uid, err := uuid.NewRandom()
	if err != nil {
		panic(errors.Wrap(err, "failed to generate random uuid (v4)"))
	}
	id := uid.String()

	return &DeviceDeployment{
		Status:         DeviceDeploymentStatusPending,
		DeviceId:       deviceId,
		DeploymentId:   deploymentId,
		Id:             id,
		Created:        &now,
		IsLogAvailable: false,
	}
}

func (d DeviceDeployment) Validate() error {
	return validation.ValidateStruct(&d,
		validation.Field(&d.Created, validation.Required),
		validation.Field(&d.Status, validation.Required),
		validation.Field(&d.DeviceId, validation.Required),
		validation.Field(&d.DeploymentId, validation.Required, is.UUID),
		validation.Field(&d.Id, validation.Required, is.UUID),
	)
}

// Deployment statistics wrapper, each value carries a count of deployments
// aggregated by state.
type Stats map[DeviceDeploymentStatus]int

func NewDeviceDeploymentStats() Stats {

	s := make(Stats)

	// populate statuses with 0s
	for _, v := range allStatuses {
		status := v.(DeviceDeploymentStatus)
		s[status] = 0
	}

	return s
}

func IsDeviceDeploymentStatusFinished(status DeviceDeploymentStatus) bool {
	if status == DeviceDeploymentStatusFailure || status == DeviceDeploymentStatusSuccess ||
		status == DeviceDeploymentStatusNoArtifact || status == DeviceDeploymentStatusAlreadyInst ||
		status == DeviceDeploymentStatusAborted || status == DeviceDeploymentStatusDecommissioned {
		return true
	}
	return false
}

// ActiveDeploymentStatuses lists statuses that represent deployment in active state (not finished).
func ActiveDeploymentStatuses() []DeviceDeploymentStatus {
	return []DeviceDeploymentStatus{
		DeviceDeploymentStatusPending,
		DeviceDeploymentStatusDownloading,
		DeviceDeploymentStatusInstalling,
		DeviceDeploymentStatusRebooting,
		DeviceDeploymentStatusPauseBeforeInstall,
		DeviceDeploymentStatusPauseBeforeCommit,
		DeviceDeploymentStatusPauseBeforeReboot,
	}
}

func InactiveDeploymentStatuses() []DeviceDeploymentStatus {
	return []DeviceDeploymentStatus{
		DeviceDeploymentStatusAlreadyInst,
		DeviceDeploymentStatusSuccess,
		DeviceDeploymentStatusFailure,
		DeviceDeploymentStatusNoArtifact,
		DeviceDeploymentStatusAlreadyInst,
		DeviceDeploymentStatusAborted,
		DeviceDeploymentStatusDecommissioned,
	}
}

// InstalledDeviceDeployment describes a deployment currently installed on the
// device, usually reported by a device
type InstalledDeviceDeployment struct {
	ArtifactName string `json:"artifact_name"`
	DeviceType   string `json:"device_type"`
}

func (i *InstalledDeviceDeployment) Validate() error {
	return validation.ValidateStruct(i,
		validation.Field(&i.ArtifactName,
			validation.Required, lengthIn1To4096,
		),
		validation.Field(&i.DeviceType,
			validation.Required, lengthIn1To4096,
		),
	)
}

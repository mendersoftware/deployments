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

	"github.com/mendersoftware/artifacts/images"
	"github.com/satori/go.uuid"
)

const (
	DeviceDeploymentStatusInProgress = "inprogress"
	DeviceDeploymentStatusPending    = "pending"
	DeviceDeploymentStatusSuccess    = "success"
	DeviceDeploymentStatusFailure    = "failure"
	DeviceDeploymentStatusNoImage    = "noimage"
)

const (
	StorageKeyDeviceDeploymentAssignedImage   = "image"
	StorageKeyDeviceDeploymentAssignedImageId = StorageKeyDeviceDeploymentAssignedImage + "." + images.StorageKeySoftwareImageId
	StorageKeyDeviceDeploymentDeviceId        = "deviceid"
	StorageKeyDeviceDeploymentStatus          = "status"
	StorageKeyDeviceDeploymentDeploymentID    = "deploymentid"
)

type DeviceDeployment struct {
	// Internal field of initial creation of deployment, required
	Created *time.Time `json:"-"`

	// Start deployment start time, optional
	Started *time.Time `json:"created,omitempty"`

	// Update finish time, optional
	Finished *time.Time `json:"finished,omitempty"`

	// Status, required, enum: "inprogress", "pending", "success", "failure"
	Status *string `json:"status"`

	// Device id, required
	DeviceId *string `json:"id"`

	// Deplyoment id
	DeploymentId *string `json:"-"`

	// ID
	Id *string `json:"-" bson:"_id"`

	// Assigned image
	Image *images.SoftwareImage `json:"-"`

	// Cache: device model
	Model *string `json:"model,omitempty"`
}

func NewDeviceDeployment(deviceId, deploymentId string) *DeviceDeployment {
	now := time.Now()
	initStatus := "pending"
	id := uuid.NewV4().String()
	return &DeviceDeployment{
		Status:       &initStatus,
		DeviceId:     &deviceId,
		DeploymentId: &deploymentId,
		Id:           &id,
		Created:      &now,
	}
}

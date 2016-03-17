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
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/artifacts/images"
	"github.com/pkg/errors"
)

const (
	ErrMsgInvalidDeployment             = "Invalid deployment"
	ErrMsgInvalidDeploymentID           = "Invalid deployment ID"
	ErrMsgStoringDeployment             = "Storing deployment"
	ErrMsgStoringDeviceDeployments      = "Storing device deployments"
	ErrMsgInvalidDeviceID               = "Invalid device ID"
	ErrMsgAssignImageToDeviceDeployment = "Assigning image to deployment for device"
	ErrMsgCheckingModelOfTheDevice      = "Checking hardware model of the device."
	ErrMsgImageUsedInActiveDeployment   = "Check if image is used by active deployment"
	ErrMsgImageUsedInDeployment         = "Check if image is used by any deployment"
	ErrMsgSearchingForDeployment        = "Searching for specified deployment"
)

type FindImageByApplicationAndModeler interface {
	FindImageByApplicationAndModel(version, model string) (*images.SoftwareImage, error)
}

type DeploymentsStorager interface {
	Insert(deployment *Deployment) error
	Delete(id string) error
	FindByID(id string) (*Deployment, error)
}

type DeviceDeploymentStorager interface {
	InsertMany(deployment ...*DeviceDeployment) error
	ExistAssignedImageWithIDAndStatuses(id string, statuses ...string) (bool, error)
}

type DeploymentsModel struct {
	imageFinder              FindImageByApplicationAndModeler
	deploymentsStorage       DeploymentsStorager
	deviceDeploymentsStorage DeviceDeploymentStorager
}

func NewDeploymentModel(
	deploymentsStorage DeploymentsStorager,
	imageFinder FindImageByApplicationAndModeler,
	deviceDeploymentsStorage DeviceDeploymentStorager,
) *DeploymentsModel {
	return &DeploymentsModel{
		imageFinder:              imageFinder,
		deploymentsStorage:       deploymentsStorage,
		deviceDeploymentsStorage: deviceDeploymentsStorage,
	}
}

func (d *DeploymentsModel) NewObject() interface{} {
	return NewDeploymentConstructor()
}

func (d *DeploymentsModel) Validate(deployment interface{}) error {
	return deployment.(*DeploymentConstructor).Validate()
}

func (d *DeploymentsModel) Create(obj interface{}) (string, error) {

	if obj == nil {
		return "", errors.New(ErrMsgInvalidDeployment)
	}

	constructorData := obj.(*DeploymentConstructor)
	deployment := NewDeploymentFromConstructor(constructorData)

	// Generate deployment for each specified device.
	deviceDeployments := make([]*DeviceDeployment, 0, len(constructorData.Devices))
	for _, id := range constructorData.Devices {

		if len(strings.TrimSpace(id)) == 0 {
			return "", errors.New(ErrMsgInvalidDeviceID)
		}

		deviceDeployment, err := d.prepareDeviceDeployment(deployment, id)
		if err != nil {
			return "", err
		}

		deviceDeployments = append(deviceDeployments, deviceDeployment)
	}

	if err := d.deploymentsStorage.Insert(deployment); err != nil {
		return "", errors.Wrap(err, ErrMsgStoringDeployment)
	}

	if err := d.deviceDeploymentsStorage.InsertMany(deviceDeployments...); err != nil {
		if errCleanup := d.deploymentsStorage.Delete(*deployment.Id); errCleanup != nil {
			err = errors.Wrap(err, errCleanup.Error())
		}

		return "", errors.Wrap(err, ErrMsgStoringDeviceDeployments)
	}

	return *deployment.Id, nil
}

func (d *DeploymentsModel) prepareDeviceDeployment(deployment *Deployment, deviceID string) (*DeviceDeployment, error) {

	model, err := d.checkModel(deviceID)
	if err != nil {
		return nil, errors.Wrap(err, ErrMsgCheckingModelOfTheDevice)
	}

	image, err := d.assignImage(*deployment.Version, model)
	if err != nil {
		return nil, errors.Wrap(err, ErrMsgAssignImageToDeviceDeployment)
	}

	deviceDeployment := NewDeviceDeployment(deviceID, *deployment.Id)
	deviceDeployment.Model = &model
	deviceDeployment.Image = image
	deviceDeployment.Created = deployment.Created

	if deviceDeployment.Image == nil {
		status := DeviceDeploymentStatusNoImage
		deviceDeployment.Status = &status
	}

	return deviceDeployment, nil
}

// TODO: This should be provided as a part of inventory service driver (dependency)
func (d *DeploymentsModel) checkModel(deviceId string) (string, error) {
	return "TestDevice", nil
}

func (d *DeploymentsModel) assignImage(version, model string) (*images.SoftwareImage, error) {
	return d.imageFinder.FindImageByApplicationAndModel(version, model)
}

// TODO: aggregre status by device (return only the worsed overall)
func (d *DeploymentsModel) GetObject(deploymentID string) (interface{}, error) {

	// Verify ID formatting
	if !govalidator.IsUUIDv4(deploymentID) {
		return nil, errors.New(ErrMsgInvalidID)
	}

	deployment, err := d.deploymentsStorage.FindByID(deploymentID)
	if err != nil {
		return nil, errors.Wrap(err, ErrMsgSearchingForDeployment)
	}

	return deployment, nil
}

// ImageUsedInActiveDeployment checks if specified image is in use by deployments
// Image is considered to be in use if it's participating in at lest one non success/error deployment.
func (d *DeploymentsModel) ImageUsedInActiveDeployment(imageId string) (bool, error) {

	// Verify ID formatting
	if !govalidator.IsUUIDv4(imageId) {
		return false, errors.New(ErrMsgInvalidID)
	}

	found, err := d.deviceDeploymentsStorage.ExistAssignedImageWithIDAndStatuses(imageId, ActiveDeploymentStatuses()...)
	if err != nil {
		return false, errors.Wrap(err, ErrMsgImageUsedInActiveDeployment)
	}

	return found, nil
}

// ImageUsedInDeployment checks if specified image is in use by deployments
// Image is considered to be in use if it's participating in any deployment.
func (d *DeploymentsModel) ImageUsedInDeployment(imageId string) (bool, error) {

	// Verify ID formatting
	if !govalidator.IsUUIDv4(imageId) {
		return false, errors.New(ErrMsgInvalidID)
	}

	found, err := d.deviceDeploymentsStorage.ExistAssignedImageWithIDAndStatuses(imageId)
	if err != nil {
		return false, errors.Wrap(err, ErrMsgImageUsedInDeployment)
	}

	return found, nil
}

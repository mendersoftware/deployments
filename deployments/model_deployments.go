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
	"github.com/pkg/errors"
)

// Defaults
const (
	DefaultUpdateDownloadLinkExpire = 24 * time.Hour
)

// Errors
var (
	ErrModelMissingInput    = errors.New("Missing input deplyoment data")
	ErrModelInvalidDeviceID = errors.New("Invalid device ID")
)

type FindImageByNameAndDeviceTyper interface {
	FindImageByNameAndDeviceType(name, model string) (*images.SoftwareImage, error)
}

type GetImageLinker interface {
	GetRequest(objectId string, duration time.Duration) (*images.Link, error)
}

type DeploymentsStorager interface {
	Insert(deployment *Deployment) error
	Delete(id string) error
	FindByID(id string) (*Deployment, error)
}

type DeviceDeploymentStorager interface {
	InsertMany(deployment ...*DeviceDeployment) error
	ExistAssignedImageWithIDAndStatuses(id string, statuses ...string) (bool, error)
	FindOldestDeploymentForDeviceIDWithStatuses(deviceID string, statuses ...string) (*DeviceDeployment, error)
}

type DeploymentsModel struct {
	imageFinder              FindImageByNameAndDeviceTyper
	deploymentsStorage       DeploymentsStorager
	deviceDeploymentsStorage DeviceDeploymentStorager
	imageLinker              GetImageLinker
}

func NewDeploymentModel(
	deploymentsStorage DeploymentsStorager,
	imageFinder FindImageByNameAndDeviceTyper,
	deviceDeploymentsStorage DeviceDeploymentStorager,
	imageLinker GetImageLinker,
) *DeploymentsModel {
	return &DeploymentsModel{
		imageFinder:              imageFinder,
		deploymentsStorage:       deploymentsStorage,
		deviceDeploymentsStorage: deviceDeploymentsStorage,
		imageLinker:              imageLinker,
	}
}

// CreateDeployment precomputes new deplyomet and schedules it for devices.
// Automatically assigns matching images to target device types.
// In case no image is available for target device, noimage status is set.
// TODO: check if specified devices are bootstrapped (when have a way to do this)
func (d *DeploymentsModel) CreateDeployment(constructor *DeploymentConstructor) (string, error) {

	if constructor == nil {
		return "", ErrModelMissingInput
	}

	if err := constructor.Validate(); err != nil {
		return "", errors.Wrap(err, "Validating deployment")
	}

	deployment := NewDeploymentFromConstructor(constructor)

	// Generate deployment for each specified device.
	deviceDeployments := make([]*DeviceDeployment, 0, len(constructor.Devices))
	for _, id := range constructor.Devices {

		deviceDeployment, err := d.prepareDeviceDeployment(deployment, id)
		if err != nil {
			return "", err
		}

		deviceDeployments = append(deviceDeployments, deviceDeployment)
	}

	if err := d.deploymentsStorage.Insert(deployment); err != nil {
		return "", errors.Wrap(err, "Storing deplyoment data")
	}

	if err := d.deviceDeploymentsStorage.InsertMany(deviceDeployments...); err != nil {
		if errCleanup := d.deploymentsStorage.Delete(*deployment.Id); errCleanup != nil {
			err = errors.Wrap(err, errCleanup.Error())
		}

		return "", errors.Wrap(err, "Storing assigned deployments to devices")
	}

	return *deployment.Id, nil
}

func (d *DeploymentsModel) prepareDeviceDeployment(deployment *Deployment, deviceID string) (*DeviceDeployment, error) {

	deviceType, err := d.checkDeviceType(deviceID)
	if err != nil {
		return nil, errors.Wrap(err, "Checking target device type")
	}

	image, err := d.assignImage(*deployment.ArtifactName, deviceType)
	if err != nil {
		return nil, errors.Wrap(err, "Assigning image targeted for device type")
	}

	deviceDeployment := NewDeviceDeployment(deviceID, *deployment.Id)
	deviceDeployment.DeviceType = &deviceType
	deviceDeployment.Image = image
	deviceDeployment.Created = deployment.Created

	// If not having appropriate image, set noimage status
	if deviceDeployment.Image == nil {
		status := DeviceDeploymentStatusNoImage
		deviceDeployment.Status = &status
	}

	return deviceDeployment, nil
}

// TODO: This should be provided as a part of inventory service driver (dependency)
func (d *DeploymentsModel) checkDeviceType(deviceID string) (string, error) {
	return "TestDevice", nil
}

func (d *DeploymentsModel) assignImage(name, deviceType string) (*images.SoftwareImage, error) {
	return d.imageFinder.FindImageByNameAndDeviceType(name, deviceType)
}

// GetDeployment fetches deplyoment by ID
func (d *DeploymentsModel) GetDeployment(deploymentID string) (*Deployment, error) {

	deployment, err := d.deploymentsStorage.FindByID(deploymentID)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for deployment by ID")
	}

	return deployment, nil
}

// ImageUsedInActiveDeployment checks if specified image is in use by deployments
// Image is considered to be in use if it's participating in at lest one non success/error deployment.
func (d *DeploymentsModel) ImageUsedInActiveDeployment(imageID string) (bool, error) {

	found, err := d.deviceDeploymentsStorage.ExistAssignedImageWithIDAndStatuses(imageID, d.ActiveDeploymentStatuses()...)
	if err != nil {
		return false, errors.Wrap(err, "Checking if image is used by active deplyoment")
	}

	return found, nil
}

// ImageUsedInDeployment checks if specified image is in use by deployments
// Image is considered to be in use if it's participating in any deployment.
func (d *DeploymentsModel) ImageUsedInDeployment(imageID string) (bool, error) {

	found, err := d.deviceDeploymentsStorage.ExistAssignedImageWithIDAndStatuses(imageID)
	if err != nil {
		return false, errors.Wrap(err, "Checking if image is used in deployment")
	}

	return found, nil
}

// GetDeploymentForDevice returns deployment for the device: currenclty still in progress or next to install.
// nil in case of nothing deploy for device.
func (d *DeploymentsModel) GetDeploymentForDevice(deviceID string) (*DeploymentInstructions, error) {

	deployment, err := d.deviceDeploymentsStorage.FindOldestDeploymentForDeviceIDWithStatuses(deviceID, d.ActiveDeploymentStatuses()...)

	if err != nil {
		return nil, errors.Wrap(err, "Searching for oldest active deployment for the device")
	}

	if deployment == nil {
		return nil, nil
	}

	link, err := d.imageLinker.GetRequest(*deployment.Image.Id, DefaultUpdateDownloadLinkExpire)
	if err != nil {
		return nil, errors.Wrap(err, "Generating download link for the device")
	}

	return NewDeploymentInstructions(*deployment.Id, link, deployment.Image), nil
}

// ActiveDeploymentStatuses lists statuses that represent deployment in active state (not finished).
func (d *DeploymentsModel) ActiveDeploymentStatuses() []string {
	return []string{
		DeviceDeploymentStatusPending,
		DeviceDeploymentStatusDownloading,
		DeviceDeploymentStatusInstalling,
		DeviceDeploymentStatusRebooting,
	}
}

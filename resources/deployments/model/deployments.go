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

package model

import (
	"time"

	"github.com/mendersoftware/deployments/resources/deployments"
	"github.com/mendersoftware/deployments/resources/deployments/controller"
	"github.com/pkg/errors"
)

// Defaults
const (
	DefaultUpdateDownloadLinkExpire = 24 * time.Hour
)

type DeploymentsModel struct {
	deploymentsStorage          DeploymentsStorage
	deviceDeploymentsStorage    DeviceDeploymentStorage
	deviceDeploymentLogsStorage DeviceDeploymentLogsStorage
	imageLinker                 GetRequester
	deviceDeploymentGenerator   Generator
}

func NewDeploymentModel(
	deploymentsStorage DeploymentsStorage,
	deviceDeploymentGenerator Generator,
	deviceDeploymentsStorage DeviceDeploymentStorage,
	deviceDeploymentLogsStorage DeviceDeploymentLogsStorage,
	imageLinker GetRequester,
) *DeploymentsModel {
	return &DeploymentsModel{
		deploymentsStorage:          deploymentsStorage,
		deviceDeploymentsStorage:    deviceDeploymentsStorage,
		deviceDeploymentLogsStorage: deviceDeploymentLogsStorage,
		imageLinker:                 imageLinker,
		deviceDeploymentGenerator:   deviceDeploymentGenerator,
	}
}

// CreateDeployment precomputes new deplyomet and schedules it for devices.
// Automatically assigns matching images to target device types.
// In case no image is available for target device, noimage status is set.
// TODO: check if specified devices are bootstrapped (when have a way to do this)
func (d *DeploymentsModel) CreateDeployment(constructor *deployments.DeploymentConstructor) (string, error) {

	if constructor == nil {
		return "", controller.ErrModelMissingInput
	}

	if err := constructor.Validate(); err != nil {
		return "", errors.Wrap(err, "Validating deployment")
	}

	deployment := deployments.NewDeploymentFromConstructor(constructor)

	// Generate deployment for each specified device.
	deviceDeployments := make([]*deployments.DeviceDeployment, 0, len(constructor.Devices))
	for _, id := range constructor.Devices {

		deviceDeployment, err := d.deviceDeploymentGenerator.Generate(id, deployment)
		if err != nil {
			return "", errors.Wrap(err, "Prepring deplyoment for device")
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

// GetDeployment fetches deplyoment by ID
func (d *DeploymentsModel) GetDeployment(deploymentID string) (*deployments.Deployment, error) {

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
func (d *DeploymentsModel) GetDeploymentForDevice(deviceID string) (*deployments.DeploymentInstructions, error) {

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

	return deployments.NewDeploymentInstructions(*deployment.DeploymentId, link, deployment.Image), nil
}

// ActiveDeploymentStatuses lists statuses that represent deployment in active state (not finished).
func (d *DeploymentsModel) ActiveDeploymentStatuses() []string {
	return []string{
		deployments.DeviceDeploymentStatusPending,
		deployments.DeviceDeploymentStatusDownloading,
		deployments.DeviceDeploymentStatusInstalling,
		deployments.DeviceDeploymentStatusRebooting,
	}
}

// UpdateDeviceDeploymentStatus will update the deployment status for device of
// ID `deviceID`. Returns nil if update was successful.
func (d *DeploymentsModel) UpdateDeviceDeploymentStatus(deploymentID string,
	deviceID string, status string) error {
	old, err := d.deviceDeploymentsStorage.UpdateDeviceDeploymentStatus(deviceID, deploymentID, status)

	if err != nil {
		return err
	}

	return d.deploymentsStorage.UpdateStats(deploymentID, old, status)
}

func (d *DeploymentsModel) GetDeploymentStats(deploymentID string) (deployments.Stats, error) {
	return d.deviceDeploymentsStorage.AggregateDeviceDeploymentByStatus(deploymentID)
}

//GetDeviceStatusesForDeployment retrieve device deployment statuses for a given deployment.
func (d *DeploymentsModel) GetDeviceStatusesForDeployment(deploymentID string) ([]deployments.DeviceDeployment, error) {
	deployment, err := d.deploymentsStorage.FindByID(deploymentID)
	if err != nil {
		return nil, controller.ErrModelInternal
	}

	if deployment == nil {
		return nil, controller.ErrModelDeploymentNotFound
	}

	statuses, err := d.deviceDeploymentsStorage.GetDeviceStatusesForDeployment(deploymentID)
	if err != nil {
		return nil, controller.ErrModelInternal
	}

	return statuses, nil
}

func (d *DeploymentsModel) LookupDeployment(query deployments.Query) ([]*deployments.Deployment, error) {
	return d.deploymentsStorage.Find(query)
}

// SaveDeviceDeploymentLog will save the deployment log for device of
// ID `deviceID`. Returns nil if log was saved successfully.
func (d *DeploymentsModel) SaveDeviceDeploymentLog(deviceID string,
	deploymentID string, logs []deployments.LogMessage) error {

	// repack to temporary deployment log and validate
	dlog := deployments.DeploymentLog{
		DeviceID:     deviceID,
		DeploymentID: deploymentID,
		Messages:     logs,
	}
	if err := dlog.Validate(); err != nil {
		return errors.Wrapf(err, controller.ErrStorageInvalidLog.Error())
	}

	if has, err := d.HasDeploymentForDevice(deploymentID, deviceID); !has {
		if err != nil {
			return err
		} else {
			return controller.ErrModelDeploymentNotFound
		}
	}

	return d.deviceDeploymentLogsStorage.SaveDeviceDeploymentLog(dlog)
}

func (d *DeploymentsModel) GetDeviceDeploymentLog(deviceID, deploymentID string) (*deployments.DeploymentLog, error) {

	return d.deviceDeploymentLogsStorage.GetDeviceDeploymentLog(deviceID, deploymentID)
}

func (d *DeploymentsModel) HasDeploymentForDevice(deploymentID string, deviceID string) (bool, error) {
	return d.deviceDeploymentsStorage.HasDeploymentForDevice(deploymentID, deviceID)
}

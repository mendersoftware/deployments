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
	"context"
	"time"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/pkg/errors"

	"github.com/mendersoftware/deployments/resources/deployments"
	"github.com/mendersoftware/deployments/resources/deployments/controller"
	"github.com/mendersoftware/deployments/resources/images"
)

// Defaults
const (
	DefaultUpdateDownloadLinkExpire = 24 * time.Hour
)

type ArtifactGetter interface {
	ImagesByName(ctx context.Context,
		artifactName string) ([]*images.SoftwareImage, error)
	ImageByIdsAndDeviceType(ctx context.Context,
		ids []string, deviceType string) (*images.SoftwareImage, error)
	ImageByNameAndDeviceType(ctx context.Context,
		name, deviceType string) (*images.SoftwareImage, error)
}

type DeploymentsModel struct {
	deploymentsStorage          DeploymentsStorage
	deviceDeploymentsStorage    DeviceDeploymentStorage
	deviceDeploymentLogsStorage DeviceDeploymentLogsStorage
	imageLinker                 GetRequester
	artifactGetter              ArtifactGetter
	imageContentType            string
}

type DeploymentsModelConfig struct {
	DeploymentsStorage          DeploymentsStorage
	DeviceDeploymentsStorage    DeviceDeploymentStorage
	DeviceDeploymentLogsStorage DeviceDeploymentLogsStorage
	ImageLinker                 GetRequester
	ArtifactGetter              ArtifactGetter
	ImageContentType            string
}

func NewDeploymentModel(config DeploymentsModelConfig) *DeploymentsModel {
	return &DeploymentsModel{
		deploymentsStorage:          config.DeploymentsStorage,
		deviceDeploymentsStorage:    config.DeviceDeploymentsStorage,
		deviceDeploymentLogsStorage: config.DeviceDeploymentLogsStorage,
		imageLinker:                 config.ImageLinker,
		artifactGetter:              config.ArtifactGetter,
		imageContentType:            config.ImageContentType,
	}
}

func getArtifactIDs(artifacts []*images.SoftwareImage) []string {
	artifactIDs := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactIDs = append(artifactIDs, artifact.Id)
	}
	return artifactIDs
}

// CreateDeployment precomputes new deplyomet and schedules it for devices.
// TODO: check if specified devices are bootstrapped (when have a way to do this)
func (d *DeploymentsModel) CreateDeployment(ctx context.Context,
	constructor *deployments.DeploymentConstructor) (string, error) {

	if constructor == nil {
		return "", controller.ErrModelMissingInput
	}

	if err := constructor.Validate(); err != nil {
		return "", errors.Wrap(err, "Validating deployment")
	}

	deployment := deployments.NewDeploymentFromConstructor(constructor)

	// Assign artifacts to the deployment.
	// Only artifacts present in the system at the moment of deployment creation
	// will be part of this deployment.
	artifacts, err := d.artifactGetter.ImagesByName(ctx, *deployment.ArtifactName)
	if err != nil {
		return "", errors.Wrap(err, "Finding artifact with given name")
	}

	if len(artifacts) == 0 {
		return "", errors.New("No artifact for deployment")
	}

	deployment.Artifacts = getArtifactIDs(artifacts)

	// Generate deployment for each specified device.
	// Do not assign artifacts to the particular device deployment.
	// Artifacts will be assigned on device update request handling, based on
	// information provided by the device in the update request.
	deviceDeployments := make([]*deployments.DeviceDeployment, 0, len(constructor.Devices))
	for _, id := range constructor.Devices {
		deviceDeployment := deployments.NewDeviceDeployment(id, *deployment.Id)
		deviceDeployment.Created = deployment.Created
		deviceDeployments = append(deviceDeployments, deviceDeployment)
	}

	// Set initial statistics cache values
	deployment.Stats[deployments.DeviceDeploymentStatusPending] = len(constructor.Devices)

	if err := d.deploymentsStorage.Insert(ctx, deployment); err != nil {
		return "", errors.Wrap(err, "Storing deployment data")
	}

	if err := d.deviceDeploymentsStorage.InsertMany(ctx, deviceDeployments...); err != nil {
		if errCleanup := d.deploymentsStorage.Delete(ctx, *deployment.Id); errCleanup != nil {
			err = errors.Wrap(err, errCleanup.Error())
		}

		return "", errors.Wrap(err, "Storing assigned deployments to devices")
	}

	return *deployment.Id, nil
}

// IsDeploymentFinished checks if there is unfinished deployment with given ID
func (d *DeploymentsModel) IsDeploymentFinished(ctx context.Context, deploymentID string) (bool, error) {

	deployment, err := d.deploymentsStorage.FindUnfinishedByID(ctx, deploymentID)
	if err != nil {
		return false, errors.Wrap(err, "Searching for unfinished deployment by ID")
	}
	if deployment == nil {
		return false, nil
	}

	return true, nil
}

// GetDeployment fetches deployment by ID
func (d *DeploymentsModel) GetDeployment(ctx context.Context,
	deploymentID string) (*deployments.Deployment, error) {

	deployment, err := d.deploymentsStorage.FindByID(ctx, deploymentID)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for deployment by ID")
	}

	return deployment, nil
}

// ImageUsedInActiveDeployment checks if specified image is in use by deployments
// Image is considered to be in use if it's participating in at lest one non success/error deployment.
func (d *DeploymentsModel) ImageUsedInActiveDeployment(ctx context.Context,
	imageID string) (bool, error) {

	var found bool

	found, err := d.deploymentsStorage.ExistUnfinishedByArtifactId(ctx, imageID)
	if err != nil {
		return false, errors.Wrap(err, "Checking if image is used by active deployment")
	}

	if found {
		return found, nil
	}

	found, err = d.deviceDeploymentsStorage.ExistAssignedImageWithIDAndStatuses(ctx,
		imageID, deployments.ActiveDeploymentStatuses()...)
	if err != nil {
		return false, errors.Wrap(err, "Checking if image is used by active deployment")
	}

	return found, nil
}

// ImageUsedInDeployment checks if specified image is in use by deployments
// Image is considered to be in use if it's participating in any deployment.
func (d *DeploymentsModel) ImageUsedInDeployment(ctx context.Context, imageID string) (bool, error) {

	var found bool

	found, err := d.deploymentsStorage.ExistUnfinishedByArtifactId(ctx, imageID)
	if err != nil {
		return false, errors.Wrap(err, "Checking if image is used by active deployment")
	}

	if found {
		return found, nil
	}

	found, err = d.deviceDeploymentsStorage.ExistAssignedImageWithIDAndStatuses(ctx, imageID)
	if err != nil {
		return false, errors.Wrap(err, "Checking if image is used in deployment")
	}

	return found, nil
}

// assignArtifact assignes artifact to the device deployment
func (d *DeploymentsModel) assignArtifact(
	ctx context.Context,
	deployment *deployments.Deployment,
	deviceDeployment *deployments.DeviceDeployment,
	installed deployments.InstalledDeviceDeployment) error {

	// Assign artifact to the device deployment.
	var artifact *images.SoftwareImage
	var err error
	// Clear device deployment image
	// New artifact will be selected for the device deployment
	// TODO: Should selecting different artifact be treated as an error?
	deviceDeployment.Image = nil

	// First case is for backward compatibility.
	// It is possible that there is old deployment structure in the system.
	// In such case we need to select artifact using name and device type.
	if deployment.Artifacts == nil || len(deployment.Artifacts) == 0 {
		artifact, err = d.artifactGetter.ImageByNameAndDeviceType(ctx, installed.Artifact, installed.DeviceType)
		if err != nil {
			return errors.Wrap(err, "assigning artifact to device deployment")
		}
	} else {
		// Select artifact for the device deployment from artifacts assgined to the deployment.
		artifact, err = d.artifactGetter.ImageByIdsAndDeviceType(ctx, deployment.Artifacts, installed.DeviceType)
		if err != nil {
			return errors.Wrap(err, "assigning artifact to device deployment")
		}
	}

	if deviceDeployment.DeploymentId == nil || deviceDeployment.DeviceId == nil {
		return controller.ErrModelInternal
	}

	// If not having appropriate image, set noartifact status
	if artifact == nil {
		if err := d.UpdateDeviceDeploymentStatus(ctx, *deviceDeployment.DeploymentId,
			*deviceDeployment.DeviceId, deployments.DeviceDeploymentStatusNoArtifact); err != nil {
			return errors.Wrap(err, "Failed to update deployment status")
		}
		return nil
	}

	if err := d.deviceDeploymentsStorage.AssignArtifact(
		ctx, *deviceDeployment.DeviceId, *deviceDeployment.DeploymentId, artifact); err != nil {
		return errors.Wrap(err, "Assigning artifact to the device deployment")
	}

	deviceDeployment.Image = artifact
	deviceDeployment.DeviceType = &installed.DeviceType

	return nil
}

// GetDeploymentForDeviceWithCurrent returns deployment for the device
func (d *DeploymentsModel) GetDeploymentForDeviceWithCurrent(ctx context.Context, deviceID string,
	installed deployments.InstalledDeviceDeployment) (*deployments.DeploymentInstructions, error) {

	deviceDeployment, err := d.deviceDeploymentsStorage.FindOldestDeploymentForDeviceIDWithStatuses(
		ctx,
		deviceID,
		deployments.ActiveDeploymentStatuses()...)

	if err != nil {
		return nil, errors.Wrap(err, "Searching for oldest active deployment for the device")
	}

	if deviceDeployment == nil {
		return nil, nil
	}

	deployment, err := d.deploymentsStorage.FindByID(ctx, *deviceDeployment.DeploymentId)
	if err != nil {
		return nil, controller.ErrModelInternal
	}

	if deployment == nil {
		return nil, nil
	}

	if installed.Artifact != "" && *deployment.ArtifactName == installed.Artifact {
		// pretend there is no deployment for this device, but update
		// its status to already installed first

		if err := d.UpdateDeviceDeploymentStatus(ctx, *deviceDeployment.DeploymentId, deviceID,
			deployments.DeviceDeploymentStatusAlreadyInst); err != nil {

			return nil, errors.Wrap(err, "Failed to update deployment status")
		}

		return nil, nil
	}

	if err := d.assignArtifact(ctx, deployment, deviceDeployment, installed); err != nil {
		return nil, err
	}

	if deviceDeployment.Image == nil {
		return nil, nil
	}

	link, err := d.imageLinker.GetRequest(ctx, deviceDeployment.Image.Id,
		DefaultUpdateDownloadLinkExpire, d.imageContentType)
	if err != nil {
		return nil, errors.Wrap(err, "Generating download link for the device")
	}

	instructions := &deployments.DeploymentInstructions{
		ID: *deviceDeployment.DeploymentId,
		Artifact: deployments.ArtifactDeploymentInstructions{
			ArtifactName:          deviceDeployment.Image.Name,
			Source:                *link,
			DeviceTypesCompatible: deviceDeployment.Image.DeviceTypesCompatible,
		},
	}

	return instructions, nil
}

// UpdateDeviceDeploymentStatus will update the deployment status for device of
// ID `deviceID`. Returns nil if update was successful.
func (d *DeploymentsModel) UpdateDeviceDeploymentStatus(ctx context.Context, deploymentID string,
	deviceID string, status string) error {

	l := log.FromContext(ctx)

	l.Infof("New status: %s for device %s deployment: %v", status, deviceID, deploymentID)

	var finishTime *time.Time = nil
	if deployments.IsDeviceDeploymentStatusFinished(status) {
		now := time.Now()
		finishTime = &now
	}

	currentStatus, err := d.deviceDeploymentsStorage.GetDeviceDeploymentStatus(ctx,
		deploymentID, deviceID)
	if err != nil {
		return err
	}

	if currentStatus == deployments.DeviceDeploymentStatusAborted {
		return controller.ErrDeploymentAborted
	}

	if currentStatus == deployments.DeviceDeploymentStatusDecommissioned {
		return controller.ErrDeviceDecommissioned
	}

	// nothing to do
	if status == currentStatus {
		return nil
	}

	old, err := d.deviceDeploymentsStorage.UpdateDeviceDeploymentStatus(ctx,
		deviceID, deploymentID,
		status, finishTime)
	if err != nil {
		return err
	}

	if err = d.deploymentsStorage.UpdateStats(ctx, deploymentID, old, status); err != nil {
		return err
	}

	// fetch deployment stats and update finished field if needed
	deployment, err := d.deploymentsStorage.FindByID(ctx, deploymentID)
	if err != nil {
		return errors.Wrap(err, "failed when searching for deployment")
	}

	if deployment.IsFinished() {
		// TODO: Make this part of UpdateStats() call as currently we are doing two
		// write operations on DB - as well as it's safer to keep them in single transaction.
		l.Infof("Finish deployment: %s", deploymentID)
		if err := d.deploymentsStorage.Finish(ctx, deploymentID, time.Now()); err != nil {
			return errors.Wrap(err, "failed to mark deployment as finished")
		}
	}

	return nil
}

func (d *DeploymentsModel) GetDeploymentStats(ctx context.Context,
	deploymentID string) (deployments.Stats, error) {

	deployment, err := d.deploymentsStorage.FindByID(ctx, deploymentID)

	if err != nil {
		return nil, errors.Wrap(err, "checking deployment id")
	}

	if deployment == nil {
		return nil, nil
	}

	return d.deviceDeploymentsStorage.AggregateDeviceDeploymentByStatus(ctx, deploymentID)
}

//GetDeviceStatusesForDeployment retrieve device deployment statuses for a given deployment.
func (d *DeploymentsModel) GetDeviceStatusesForDeployment(ctx context.Context,
	deploymentID string) ([]deployments.DeviceDeployment, error) {

	deployment, err := d.deploymentsStorage.FindByID(ctx, deploymentID)
	if err != nil {
		return nil, controller.ErrModelInternal
	}

	if deployment == nil {
		return nil, controller.ErrModelDeploymentNotFound
	}

	statuses, err := d.deviceDeploymentsStorage.GetDeviceStatusesForDeployment(ctx, deploymentID)
	if err != nil {
		return nil, controller.ErrModelInternal
	}

	return statuses, nil
}

func (d *DeploymentsModel) LookupDeployment(ctx context.Context,
	query deployments.Query) ([]*deployments.Deployment, error) {
	list, err := d.deploymentsStorage.Find(ctx, query)

	if err != nil {
		return nil, errors.Wrap(err, "searching for deployments")
	}

	if list == nil {
		return make([]*deployments.Deployment, 0), nil
	}

	return list, nil
}

// SaveDeviceDeploymentLog will save the deployment log for device of
// ID `deviceID`. Returns nil if log was saved successfully.
func (d *DeploymentsModel) SaveDeviceDeploymentLog(ctx context.Context, deviceID string,
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

	if has, err := d.HasDeploymentForDevice(ctx, deploymentID, deviceID); !has {
		if err != nil {
			return err
		} else {
			return controller.ErrModelDeploymentNotFound
		}
	}

	if err := d.deviceDeploymentLogsStorage.SaveDeviceDeploymentLog(ctx, dlog); err != nil {
		return err
	}

	return d.deviceDeploymentsStorage.UpdateDeviceDeploymentLogAvailability(ctx,
		deviceID, deploymentID, true)
}

func (d *DeploymentsModel) GetDeviceDeploymentLog(ctx context.Context,
	deviceID, deploymentID string) (*deployments.DeploymentLog, error) {

	return d.deviceDeploymentLogsStorage.GetDeviceDeploymentLog(ctx,
		deviceID, deploymentID)
}

func (d *DeploymentsModel) HasDeploymentForDevice(ctx context.Context,
	deploymentID string, deviceID string) (bool, error) {
	return d.deviceDeploymentsStorage.HasDeploymentForDevice(ctx, deploymentID, deviceID)
}

// AbortDeployment aborts deployment for devices and updates deployment stats
func (d *DeploymentsModel) AbortDeployment(ctx context.Context, deploymentID string) error {

	if err := d.deviceDeploymentsStorage.AbortDeviceDeployments(ctx, deploymentID); err != nil {
		return err
	}

	stats, err := d.deviceDeploymentsStorage.AggregateDeviceDeploymentByStatus(
		ctx, deploymentID)
	if err != nil {
		return err
	}

	// Update deployment stats and finish deployment (set finished timestamp to current time)
	// Aborted deployment is considered to be finished even if some devices are
	// still processing this deployment.
	return d.deploymentsStorage.UpdateStatsAndFinishDeployment(ctx,
		deploymentID, stats)
}

func (d *DeploymentsModel) DecommissionDevice(ctx context.Context, deviceId string) error {

	if err := d.deviceDeploymentsStorage.DecommissionDeviceDeployments(ctx,
		deviceId); err != nil {

		return err
	}

	//get all affected deployments and update its stats
	deviceDeployments, err := d.deviceDeploymentsStorage.FindAllDeploymentsForDeviceIDWithStatuses(
		ctx,
		deviceId, deployments.DeviceDeploymentStatusDecommissioned)

	if err != nil {
		return err
	}

	for _, deviceDeployment := range deviceDeployments {

		stats, err := d.deviceDeploymentsStorage.AggregateDeviceDeploymentByStatus(
			ctx, *deviceDeployment.DeploymentId)
		if err != nil {
			return err
		}
		if err := d.deploymentsStorage.UpdateStatsAndFinishDeployment(
			ctx, *deviceDeployment.DeploymentId, stats); err != nil {
			return err
		}
	}

	return nil
}

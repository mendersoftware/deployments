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

package app

import (
	"context"
	"io"
	"io/ioutil"
	"time"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/mender-artifact/areader"
	"github.com/mendersoftware/mender-artifact/artifact"
	"github.com/mendersoftware/mender-artifact/handlers"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/s3"
	"github.com/mendersoftware/deployments/store"
	"github.com/mendersoftware/deployments/store/mongo"
)

const (
	ArtifactContentType = "application/vnd.mender-artifact"

	DefaultUpdateDownloadLinkExpire = 24 * time.Hour
)

// Errors expected from App interface
var (
	// images
	ErrImageMetaNotFound                = errors.New("Image metadata is not found")
	ErrModelMultipartUploadMsgMalformed = errors.New("Multipart upload message malformed")
	ErrModelMissingInputMetadata        = errors.New("Missing input metadata")
	ErrModelMissingInputArtifact        = errors.New("Missing input artifact")
	ErrModelInvalidMetadata             = errors.New("Metadata invalid")
	ErrModelArtifactNotUnique           = errors.New("Artifact not unique")
	ErrModelArtifactFileTooLarge        = errors.New("Artifact file too large")
	ErrModelImageInActiveDeployment     = errors.New("Image is used in active deployment and cannot be removed")
	ErrModelImageUsedInAnyDeployment    = errors.New("Image has already been used in deployment")
	ErrModelParsingArtifactFailed       = errors.New("Cannot parse artifact file")

	// deployments
	ErrModelMissingInput       = errors.New("Missing input deployment data")
	ErrModelInvalidDeviceID    = errors.New("Invalid device ID")
	ErrModelDeploymentNotFound = errors.New("Deployment not found")
	ErrModelInternal           = errors.New("Internal error")
	ErrStorageInvalidLog       = errors.New("Invalid deployment log")
	ErrStorageNotFound         = errors.New("Not found")
	ErrDeploymentAborted       = errors.New("Deployment aborted")
	ErrDeviceDecommissioned    = errors.New("Device decommissioned")
	ErrNoArtifact              = errors.New("No artifact for the deployment")
)

//deployments

type App interface {
	// limits
	GetLimit(ctx context.Context, name string) (*model.Limit, error)
	ProvisionTenant(ctx context.Context, tenant_id string) error

	// images
	ListImages(ctx context.Context,
		filters map[string]string) ([]*model.SoftwareImage, error)
	DownloadLink(ctx context.Context, imageID string,
		expire time.Duration) (*model.Link, error)
	GetImage(ctx context.Context, id string) (*model.SoftwareImage, error)
	DeleteImage(ctx context.Context, imageID string) error
	CreateImage(ctx context.Context,
		multipartUploadMsg *model.MultipartUploadMsg) (string, error)
	EditImage(ctx context.Context, id string,
		constructorData *model.SoftwareImageMetaConstructor) (bool, error)

	// deployments
	CreateDeployment(ctx context.Context,
		constructor *model.DeploymentConstructor) (string, error)
	GetDeployment(ctx context.Context, deploymentID string) (*model.Deployment, error)
	IsDeploymentFinished(ctx context.Context, deploymentID string) (bool, error)
	AbortDeployment(ctx context.Context, deploymentID string) error
	GetDeploymentStats(ctx context.Context, deploymentID string) (model.Stats, error)
	GetDeploymentForDeviceWithCurrent(ctx context.Context, deviceID string,
		current model.InstalledDeviceDeployment) (*model.DeploymentInstructions, error)
	HasDeploymentForDevice(ctx context.Context, deploymentID string,
		deviceID string) (bool, error)
	UpdateDeviceDeploymentStatus(ctx context.Context, deploymentID string,
		deviceID string, status model.DeviceDeploymentStatus) error
	GetDeviceStatusesForDeployment(ctx context.Context,
		deploymentID string) ([]model.DeviceDeployment, error)
	LookupDeployment(ctx context.Context,
		query model.Query) ([]*model.Deployment, error)
	SaveDeviceDeploymentLog(ctx context.Context, deviceID string,
		deploymentID string, logs []model.LogMessage) error
	GetDeviceDeploymentLog(ctx context.Context,
		deviceID, deploymentID string) (*model.DeploymentLog, error)
	DecommissionDevice(ctx context.Context, deviceID string) error
}

type Deployments struct {
	db               store.DataStore
	fileStorage      s3.FileStorage
	imageContentType string
}

func NewDeployments(storage store.DataStore, fileStorage s3.FileStorage, imageContentType string) *Deployments {
	return &Deployments{
		db:               storage,
		fileStorage:      fileStorage,
		imageContentType: imageContentType,
	}
}

func (d *Deployments) GetLimit(ctx context.Context, name string) (*model.Limit, error) {
	limit, err := d.db.GetLimit(ctx, name)
	if err == mongo.ErrLimitNotFound {
		return &model.Limit{
			Name:  name,
			Value: 0,
		}, nil

	} else if err != nil {
		return nil, errors.Wrap(err, "failed to obtain limit from storage")
	}
	return limit, nil
}

func (d *Deployments) ProvisionTenant(ctx context.Context, tenant_id string) error {
	if err := d.db.ProvisionTenant(ctx, tenant_id); err != nil {
		return errors.Wrap(err, "failed to provision tenant")
	}

	return nil
}

// CreateImage parses artifact and uploads artifact file to the file storage - in parallel,
// and creates image structure in the system.
// Returns image ID and nil on success.
func (d *Deployments) CreateImage(ctx context.Context,
	multipartUploadMsg *model.MultipartUploadMsg) (string, error) {

	// maximum image size is 10G
	const MaxImageSize = 1024 * 1024 * 1024 * 10

	switch {
	case multipartUploadMsg == nil:
		return "", ErrModelMultipartUploadMsgMalformed
	case multipartUploadMsg.MetaConstructor == nil:
		return "", ErrModelMissingInputMetadata
	case multipartUploadMsg.ArtifactReader == nil:
		return "", ErrModelMissingInputArtifact
	case multipartUploadMsg.ArtifactSize > MaxImageSize:
		return "", ErrModelArtifactFileTooLarge
	}

	artifactID, err := d.handleArtifact(ctx, multipartUploadMsg)
	// try to remove artifact file from file storage on error
	if err != nil {
		if cleanupErr := d.fileStorage.Delete(ctx,
			artifactID); cleanupErr != nil {
			return "", errors.Wrap(err, cleanupErr.Error())
		}
	}
	return artifactID, err
}

// handleArtifact parses artifact and uploads artifact file to the file storage - in parallel,
// and creates image structure in the system.
// Returns image ID, artifact file ID and nil on success.
func (d *Deployments) handleArtifact(ctx context.Context,
	multipartUploadMsg *model.MultipartUploadMsg) (string, error) {

	// create pipe
	pR, pW := io.Pipe()

	// limit reader to the size provided with the upload message
	lr := io.LimitReader(multipartUploadMsg.ArtifactReader, multipartUploadMsg.ArtifactSize)
	tee := io.TeeReader(lr, pW)

	uid, err := uuid.NewV4()
	if err != nil {
		return "", errors.New("failed to generate new uuid")
	}

	artifactID := uid.String()

	ch := make(chan error)
	// create goroutine for artifact upload
	//
	// reading from the pipe (which is done in UploadArtifact method) is a blocking operation
	// and cannot be done in the same goroutine as writing to the pipe
	//
	// uploading and parsing artifact in the same process will cause in a deadlock!
	go func() {
		err := d.fileStorage.UploadArtifact(ctx,
			artifactID, multipartUploadMsg.ArtifactSize, pR, ArtifactContentType)
		if err != nil {
			pR.CloseWithError(err)
		}
		ch <- err
	}()

	// parse artifact
	// artifact library reads all the data from the given reader
	metaArtifactConstructor, err := getMetaFromArchive(&tee)
	if err != nil {
		pW.Close()
		<-ch
		return artifactID, ErrModelParsingArtifactFailed
	}

	// read the rest of the data,
	// just in case the artifact library did not read all the data from the reader
	_, err = io.Copy(ioutil.Discard, tee)
	if err != nil {
		pW.Close()
		<-ch
		return artifactID, err
	}

	// close the pipe
	pW.Close()

	// collect output from the goroutine
	if uploadResponseErr := <-ch; uploadResponseErr != nil {
		return artifactID, uploadResponseErr
	}

	// validate artifact metadata
	if err = metaArtifactConstructor.Validate(); err != nil {
		return artifactID, ErrModelInvalidMetadata
	}

	// check if artifact is unique
	// artifact is considered to be unique if there is no artifact with the same name
	// and supporing the same platform in the system
	isArtifactUnique, err := d.db.IsArtifactUnique(ctx,
		metaArtifactConstructor.Name, metaArtifactConstructor.DeviceTypesCompatible)
	if err != nil {
		return artifactID, errors.Wrap(err, "Fail to check if artifact is unique")
	}
	if !isArtifactUnique {
		return artifactID, ErrModelArtifactNotUnique
	}

	image := model.NewSoftwareImage(
		artifactID, multipartUploadMsg.MetaConstructor, metaArtifactConstructor, multipartUploadMsg.ArtifactSize)

	// save image structure in the system
	if err = d.db.InsertImage(ctx, image); err != nil {
		return artifactID, errors.Wrap(err, "Fail to store the metadata")
	}

	return artifactID, nil
}

// GetImage allows to fetch image obeject with specified id
// Nil if not found
func (d *Deployments) GetImage(ctx context.Context, id string) (*model.SoftwareImage, error) {

	image, err := d.db.FindImageByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image with specified ID")
	}

	if image == nil {
		return nil, nil
	}

	return image, nil
}

// DeleteImage removes metadata and image file
// Noop for not exisitng images
// Allowed to remove image only if image is not scheduled or in progress for an updates - then image file is needed
// In case of already finished updates only image file is not needed, metadata is attached directly to device deployment
// therefore we still have some information about image that have been used (but not the file)
func (d *Deployments) DeleteImage(ctx context.Context, imageID string) error {
	found, err := d.GetImage(ctx, imageID)

	if err != nil {
		return errors.Wrap(err, "Getting image metadata")
	}

	if found == nil {
		return ErrImageMetaNotFound
	}

	inUse, err := d.ImageUsedInActiveDeployment(ctx, imageID)
	if err != nil {
		return errors.Wrap(err, "Checking if image is used in active deployment")
	}

	// Image is in use, not allowed to delete
	if inUse {
		return ErrModelImageInActiveDeployment
	}

	// Delete image file (call to external service)
	// Noop for not existing file
	if err := d.fileStorage.Delete(ctx, imageID); err != nil {
		return errors.Wrap(err, "Deleting image file")
	}

	// Delete metadata
	if err := d.db.DeleteImage(ctx, imageID); err != nil {
		return errors.Wrap(err, "Deleting image metadata")
	}

	return nil
}

// ListImages according to specified filers.
func (d *Deployments) ListImages(ctx context.Context,
	filters map[string]string) ([]*model.SoftwareImage, error) {

	imageList, err := d.db.FindAll(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image metadata")
	}

	if imageList == nil {
		return make([]*model.SoftwareImage, 0), nil
	}

	return imageList, nil
}

// EditObject allows editing only if image have not been used yet in any deployment.
func (d *Deployments) EditImage(ctx context.Context, imageID string,
	constructor *model.SoftwareImageMetaConstructor) (bool, error) {

	if err := constructor.Validate(); err != nil {
		return false, errors.Wrap(err, "Validating image metadata")
	}

	found, err := d.ImageUsedInDeployment(ctx, imageID)
	if err != nil {
		return false, errors.Wrap(err, "Searching for usage of the image among deployments")
	}

	if found {
		return false, ErrModelImageUsedInAnyDeployment
	}

	foundImage, err := d.db.FindImageByID(ctx, imageID)
	if err != nil {
		return false, errors.Wrap(err, "Searching for image with specified ID")
	}

	if foundImage == nil {
		return false, nil
	}

	foundImage.SetModified(time.Now())
	foundImage.SoftwareImageMetaConstructor = *constructor

	_, err = d.db.Update(ctx, foundImage)
	if err != nil {
		return false, errors.Wrap(err, "Updating image matadata")
	}

	return true, nil
}

// DownloadLink presigned GET link to download image file.
// Returns error if image have not been uploaded.
func (d *Deployments) DownloadLink(ctx context.Context, imageID string,
	expire time.Duration) (*model.Link, error) {

	found, err := d.db.Exists(ctx, imageID)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image with specified ID")
	}

	if !found {
		return nil, nil
	}

	found, err = d.fileStorage.Exists(ctx, imageID)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image file")
	}

	if !found {
		return nil, nil
	}

	link, err := d.fileStorage.GetRequest(ctx, imageID,
		expire, ArtifactContentType)
	if err != nil {
		return nil, errors.Wrap(err, "Generating download link")
	}

	return link, nil
}

func getArtifactInfo(info artifact.Info) *model.ArtifactInfo {
	return &model.ArtifactInfo{
		Format:  info.Format,
		Version: uint(info.Version),
	}
}

func getUpdateFiles(uFiles []*handlers.DataFile) ([]model.UpdateFile, error) {
	var files []model.UpdateFile
	for _, u := range uFiles {
		files = append(files, model.UpdateFile{
			Name:     u.Name,
			Size:     u.Size,
			Date:     &u.Date,
			Checksum: string(u.Checksum),
		})
	}
	return files, nil
}

func getMetaFromArchive(r *io.Reader) (*model.SoftwareImageMetaArtifactConstructor, error) {
	metaArtifact := model.NewSoftwareImageMetaArtifactConstructor()

	aReader := areader.NewReader(*r)

	// There is no signature verification here.
	// It is just simple check if artifact is signed or not.
	aReader.VerifySignatureCallback = func(message, sig []byte) error {
		metaArtifact.Signed = true
		return nil
	}

	err := aReader.ReadArtifact()
	if err != nil {
		return nil, errors.Wrap(err, "reading artifact error")
	}

	metaArtifact.Info = getArtifactInfo(aReader.GetInfo())
	metaArtifact.DeviceTypesCompatible = aReader.GetCompatibleDevices()
	metaArtifact.Name = aReader.GetArtifactName()

	for _, p := range aReader.GetHandlers() {
		uFiles, err := getUpdateFiles(p.GetUpdateFiles())
		if err != nil {
			return nil, errors.Wrap(err, "Cannot get update files:")
		}

		uMetadata, err := p.GetUpdateMetaData()
		if err != nil {
			return nil, errors.Wrap(err, "Cannot get update metadata")
		}

		metaArtifact.Updates = append(
			metaArtifact.Updates,
			model.Update{
				TypeInfo: model.ArtifactUpdateTypeInfo{
					Type: p.GetUpdateType(),
				},
				Files:    uFiles,
				MetaData: uMetadata,
			})
	}

	return metaArtifact, nil
}

func getArtifactIDs(artifacts []*model.SoftwareImage) []string {
	artifactIDs := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactIDs = append(artifactIDs, artifact.Id)
	}
	return artifactIDs
}

// deployments

// CreateDeployment precomputes new deplyomet and schedules it for devices.
// TODO: check if specified devices are bootstrapped (when have a way to do this)
func (d *Deployments) CreateDeployment(ctx context.Context,
	constructor *model.DeploymentConstructor) (string, error) {

	if constructor == nil {
		return "", ErrModelMissingInput
	}

	if err := constructor.Validate(); err != nil {
		return "", errors.Wrap(err, "Validating deployment")
	}

	deployment, err := model.NewDeploymentFromConstructor(constructor)
	if err != nil {
		return "", errors.Wrap(err, "failed to create deployment")
	}

	// Assign artifacts to the deployment.
	// Only artifacts present in the system at the moment of deployment creation
	// will be part of this deployment.
	artifacts, err := d.db.ImagesByName(ctx, *deployment.ArtifactName)
	if err != nil {
		return "", errors.Wrap(err, "Finding artifact with given name")
	}

	if len(artifacts) == 0 {
		return "", ErrNoArtifact
	}

	deployment.Artifacts = getArtifactIDs(artifacts)

	// Generate deployment for each specified device.
	// Do not assign artifacts to the particular device deployment.
	// Artifacts will be assigned on device update request handling, based on
	// information provided by the device in the update request.
	deviceDeployments := make([]*model.DeviceDeployment, 0, len(constructor.Devices))
	for _, id := range constructor.Devices {
		deviceDeployment, err := model.NewDeviceDeployment(id, *deployment.Id)
		if err != nil {
			return "", errors.Wrap(err, "failed to create device deployment")
		}

		deviceDeployment.Created = deployment.Created
		deviceDeployments = append(deviceDeployments, deviceDeployment)
	}

	// Set initial statistics cache values
	deployment.Stats[model.DeviceDeploymentStatusPending] = len(constructor.Devices)

	if err := d.db.InsertDeployment(ctx, deployment); err != nil {
		return "", errors.Wrap(err, "Storing deployment data")
	}

	if err := d.db.InsertMany(ctx, deviceDeployments...); err != nil {
		if errCleanup := d.db.DeleteDeployment(ctx, *deployment.Id); errCleanup != nil {
			err = errors.Wrap(err, errCleanup.Error())
		}

		return "", errors.Wrap(err, "Storing assigned deployments to devices")
	}

	return *deployment.Id, nil
}

// IsDeploymentFinished checks if there is unfinished deployment with given ID
func (d *Deployments) IsDeploymentFinished(ctx context.Context, deploymentID string) (bool, error) {

	deployment, err := d.db.FindUnfinishedByID(ctx, deploymentID)
	if err != nil {
		return false, errors.Wrap(err, "Searching for unfinished deployment by ID")
	}
	if deployment == nil {
		return true, nil
	}

	return false, nil
}

// GetDeployment fetches deployment by ID
func (d *Deployments) GetDeployment(ctx context.Context,
	deploymentID string) (*model.Deployment, error) {

	deployment, err := d.db.FindDeploymentByID(ctx, deploymentID)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for deployment by ID")
	}

	return deployment, nil
}

// ImageUsedInActiveDeployment checks if specified image is in use by deployments
// Image is considered to be in use if it's participating in at lest one non success/error deployment.
func (d *Deployments) ImageUsedInActiveDeployment(ctx context.Context,
	imageID string) (bool, error) {

	var found bool

	found, err := d.db.ExistUnfinishedByArtifactId(ctx, imageID)
	if err != nil {
		return false, errors.Wrap(err, "Checking if image is used by active deployment")
	}

	if found {
		return found, nil
	}

	found, err = d.db.ExistAssignedImageWithIDAndStatuses(ctx,
		imageID, model.ActiveDeploymentStatuses()...)
	if err != nil {
		return false, errors.Wrap(err, "Checking if image is used by active deployment")
	}

	return found, nil
}

// ImageUsedInDeployment checks if specified image is in use by deployments
// Image is considered to be in use if it's participating in any deployment.
func (d *Deployments) ImageUsedInDeployment(ctx context.Context, imageID string) (bool, error) {

	var found bool

	found, err := d.db.ExistUnfinishedByArtifactId(ctx, imageID)
	if err != nil {
		return false, errors.Wrap(err, "Checking if image is used by active deployment")
	}

	if found {
		return found, nil
	}

	found, err = d.db.ExistAssignedImageWithIDAndStatuses(ctx, imageID)
	if err != nil {
		return false, errors.Wrap(err, "Checking if image is used in deployment")
	}

	return found, nil
}

// assignArtifact assignes artifact to the device deployment
func (d *Deployments) assignArtifact(
	ctx context.Context,
	deployment *model.Deployment,
	deviceDeployment *model.DeviceDeployment,
	installed model.InstalledDeviceDeployment) error {

	// Assign artifact to the device deployment.
	var artifact *model.SoftwareImage
	var err error
	// Clear device deployment image
	// New artifact will be selected for the device deployment
	// TODO: Should selecting different artifact be treated as an error?
	deviceDeployment.Image = nil

	// First case is for backward compatibility.
	// It is possible that there is old deployment structure in the system.
	// In such case we need to select artifact using name and device type.
	if deployment.Artifacts == nil || len(deployment.Artifacts) == 0 {
		artifact, err = d.db.ImageByNameAndDeviceType(ctx, installed.Artifact, installed.DeviceType)
		if err != nil {
			return errors.Wrap(err, "assigning artifact to device deployment")
		}
	} else {
		// Select artifact for the device deployment from artifacts assgined to the deployment.
		artifact, err = d.db.ImageByIdsAndDeviceType(ctx, deployment.Artifacts, installed.DeviceType)
		if err != nil {
			return errors.Wrap(err, "assigning artifact to device deployment")
		}
	}

	if deviceDeployment.DeploymentId == nil || deviceDeployment.DeviceId == nil {
		return ErrModelInternal
	}

	// If not having appropriate image, set noartifact status
	if artifact == nil {
		if err := d.UpdateDeviceDeploymentStatus(ctx, *deviceDeployment.DeploymentId,
			*deviceDeployment.DeviceId,
			model.DeviceDeploymentStatus{
				Status: model.DeviceDeploymentStatusNoArtifact,
			}); err != nil {
			return errors.Wrap(err, "Failed to update deployment status")
		}
		return nil
	}

	if err := d.db.AssignArtifact(
		ctx, *deviceDeployment.DeviceId, *deviceDeployment.DeploymentId, artifact); err != nil {
		return errors.Wrap(err, "Assigning artifact to the device deployment")
	}

	deviceDeployment.Image = artifact
	deviceDeployment.DeviceType = &installed.DeviceType

	return nil
}

// GetDeploymentForDeviceWithCurrent returns deployment for the device
func (d *Deployments) GetDeploymentForDeviceWithCurrent(ctx context.Context, deviceID string,
	installed model.InstalledDeviceDeployment) (*model.DeploymentInstructions, error) {

	deviceDeployment, err := d.db.FindOldestDeploymentForDeviceIDWithStatuses(
		ctx,
		deviceID,
		model.ActiveDeploymentStatuses()...)

	if err != nil {
		return nil, errors.Wrap(err, "Searching for oldest active deployment for the device")
	}

	if deviceDeployment == nil {
		return nil, nil
	}

	deployment, err := d.db.FindDeploymentByID(ctx, *deviceDeployment.DeploymentId)
	if err != nil {
		return nil, ErrModelInternal
	}

	if deployment == nil {
		return nil, nil
	}

	if installed.Artifact != "" && *deployment.ArtifactName == installed.Artifact {
		// pretend there is no deployment for this device, but update
		// its status to already installed first

		if err := d.UpdateDeviceDeploymentStatus(ctx, *deviceDeployment.DeploymentId, deviceID,
			model.DeviceDeploymentStatus{
				Status: model.DeviceDeploymentStatusAlreadyInst,
			}); err != nil {

			return nil, errors.Wrap(err, "Failed to update deployment status")
		}

		return nil, nil
	}

	// assign artifact only if the artifact was not assigned previously or the device type has changed
	if deviceDeployment.Image == nil || deviceDeployment.DeviceType == nil || *deviceDeployment.DeviceType != installed.DeviceType {
		if err := d.assignArtifact(ctx, deployment, deviceDeployment, installed); err != nil {
			return nil, err
		}
	}

	if deviceDeployment.Image == nil {
		return nil, nil
	}

	link, err := d.fileStorage.GetRequest(ctx, deviceDeployment.Image.Id,
		DefaultUpdateDownloadLinkExpire, d.imageContentType)
	if err != nil {
		return nil, errors.Wrap(err, "Generating download link for the device")
	}

	instructions := &model.DeploymentInstructions{
		ID: *deviceDeployment.DeploymentId,
		Artifact: model.ArtifactDeploymentInstructions{
			ArtifactName:          deviceDeployment.Image.Name,
			Source:                *link,
			DeviceTypesCompatible: deviceDeployment.Image.DeviceTypesCompatible,
		},
	}

	return instructions, nil
}

// UpdateDeviceDeploymentStatus will update the deployment status for device of
// ID `deviceID`. Returns nil if update was successful.
func (d *Deployments) UpdateDeviceDeploymentStatus(ctx context.Context, deploymentID string,
	deviceID string, ddStatus model.DeviceDeploymentStatus) error {

	l := log.FromContext(ctx)

	l.Infof("New status: %s for device %s deployment: %v", ddStatus.Status, deviceID, deploymentID)

	var finishTime *time.Time = nil
	if model.IsDeviceDeploymentStatusFinished(ddStatus.Status) {
		now := time.Now()
		finishTime = &now
	}

	currentStatus, err := d.db.GetDeviceDeploymentStatus(ctx,
		deploymentID, deviceID)
	if err != nil {
		return err
	}

	if currentStatus == model.DeviceDeploymentStatusAborted {
		return ErrDeploymentAborted
	}

	if currentStatus == model.DeviceDeploymentStatusDecommissioned {
		return ErrDeviceDecommissioned
	}

	// nothing to do
	if ddStatus.Status == currentStatus {
		return nil
	}

	// update finish time
	ddStatus.FinishTime = finishTime

	old, err := d.db.UpdateDeviceDeploymentStatus(ctx,
		deviceID, deploymentID, ddStatus)
	if err != nil {
		return err
	}

	if err = d.db.UpdateStats(ctx, deploymentID, old, ddStatus.Status); err != nil {
		return err
	}

	// fetch deployment stats and update finished field if needed
	deployment, err := d.db.FindDeploymentByID(ctx, deploymentID)
	if err != nil {
		return errors.Wrap(err, "failed when searching for deployment")
	}

	if deployment.IsFinished() {
		// TODO: Make this part of UpdateStats() call as currently we are doing two
		// write operations on DB - as well as it's safer to keep them in single transaction.
		l.Infof("Finish deployment: %s", deploymentID)
		if err := d.db.Finish(ctx, deploymentID, time.Now()); err != nil {
			return errors.Wrap(err, "failed to mark deployment as finished")
		}
	}

	return nil
}

func (d *Deployments) GetDeploymentStats(ctx context.Context,
	deploymentID string) (model.Stats, error) {

	deployment, err := d.db.FindDeploymentByID(ctx, deploymentID)

	if err != nil {
		return nil, errors.Wrap(err, "checking deployment id")
	}

	if deployment == nil {
		return nil, nil
	}

	return d.db.AggregateDeviceDeploymentByStatus(ctx, deploymentID)
}

//GetDeviceStatusesForDeployment retrieve device deployment statuses for a given deployment.
func (d *Deployments) GetDeviceStatusesForDeployment(ctx context.Context,
	deploymentID string) ([]model.DeviceDeployment, error) {

	deployment, err := d.db.FindDeploymentByID(ctx, deploymentID)
	if err != nil {
		return nil, ErrModelInternal
	}

	if deployment == nil {
		return nil, ErrModelDeploymentNotFound
	}

	statuses, err := d.db.GetDeviceStatusesForDeployment(ctx, deploymentID)
	if err != nil {
		return nil, ErrModelInternal
	}

	return statuses, nil
}

func (d *Deployments) LookupDeployment(ctx context.Context,
	query model.Query) ([]*model.Deployment, error) {
	list, err := d.db.Find(ctx, query)

	if err != nil {
		return nil, errors.Wrap(err, "searching for deployments")
	}

	if list == nil {
		return make([]*model.Deployment, 0), nil
	}

	for _, deployment := range list {
		if deviceCount, err := d.db.DeviceCountByDeployment(ctx,
			*deployment.Id); err != nil {
			return nil, errors.Wrap(err, "counting device deployments")
		} else {
			deployment.DeviceCount = deviceCount
		}
	}

	return list, nil
}

// SaveDeviceDeploymentLog will save the deployment log for device of
// ID `deviceID`. Returns nil if log was saved successfully.
func (d *Deployments) SaveDeviceDeploymentLog(ctx context.Context, deviceID string,
	deploymentID string, logs []model.LogMessage) error {

	// repack to temporary deployment log and validate
	dlog := model.DeploymentLog{
		DeviceID:     deviceID,
		DeploymentID: deploymentID,
		Messages:     logs,
	}
	if err := dlog.Validate(); err != nil {
		return errors.Wrapf(err, ErrStorageInvalidLog.Error())
	}

	if has, err := d.HasDeploymentForDevice(ctx, deploymentID, deviceID); !has {
		if err != nil {
			return err
		} else {
			return ErrModelDeploymentNotFound
		}
	}

	if err := d.db.SaveDeviceDeploymentLog(ctx, dlog); err != nil {
		return err
	}

	return d.db.UpdateDeviceDeploymentLogAvailability(ctx,
		deviceID, deploymentID, true)
}

func (d *Deployments) GetDeviceDeploymentLog(ctx context.Context,
	deviceID, deploymentID string) (*model.DeploymentLog, error) {

	return d.db.GetDeviceDeploymentLog(ctx,
		deviceID, deploymentID)
}

func (d *Deployments) HasDeploymentForDevice(ctx context.Context,
	deploymentID string, deviceID string) (bool, error) {
	return d.db.HasDeploymentForDevice(ctx, deploymentID, deviceID)
}

// AbortDeployment aborts deployment for devices and updates deployment stats
func (d *Deployments) AbortDeployment(ctx context.Context, deploymentID string) error {

	if err := d.db.AbortDeviceDeployments(ctx, deploymentID); err != nil {
		return err
	}

	stats, err := d.db.AggregateDeviceDeploymentByStatus(
		ctx, deploymentID)
	if err != nil {
		return err
	}

	// Update deployment stats and finish deployment (set finished timestamp to current time)
	// Aborted deployment is considered to be finished even if some devices are
	// still processing this deployment.
	return d.db.UpdateStatsAndFinishDeployment(ctx,
		deploymentID, stats)
}

func (d *Deployments) DecommissionDevice(ctx context.Context, deviceId string) error {

	if err := d.db.DecommissionDeviceDeployments(ctx,
		deviceId); err != nil {

		return err
	}

	//get all affected deployments and update its stats
	deviceDeployments, err := d.db.FindAllDeploymentsForDeviceIDWithStatuses(
		ctx,
		deviceId, model.DeviceDeploymentStatusDecommissioned)

	if err != nil {
		return err
	}

	for _, deviceDeployment := range deviceDeployments {

		stats, err := d.db.AggregateDeviceDeploymentByStatus(
			ctx, *deviceDeployment.DeploymentId)
		if err != nil {
			return err
		}
		if err := d.db.UpdateStatsAndFinishDeployment(
			ctx, *deviceDeployment.DeploymentId, stats); err != nil {
			return err
		}
	}

	return nil
}

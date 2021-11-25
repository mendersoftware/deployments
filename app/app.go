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

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/mender-artifact/areader"
	"github.com/mendersoftware/mender-artifact/artifact"
	"github.com/mendersoftware/mender-artifact/awriter"
	"github.com/mendersoftware/mender-artifact/handlers"

	"github.com/mendersoftware/deployments/client/inventory"
	"github.com/mendersoftware/deployments/client/reporting"
	"github.com/mendersoftware/deployments/client/workflows"
	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/s3"
	"github.com/mendersoftware/deployments/store"
	"github.com/mendersoftware/deployments/store/mongo"
	"github.com/mendersoftware/deployments/utils"
)

const (
	ArtifactContentType              = "application/vnd.mender-artifact"
	ArtifactConfigureType            = "mender-configure"
	ArtifactConfigureProvides        = "data-partition.mender-configure.version"
	ArtifactConfigureProvidesCleared = "data-partition.mender-configure.*"

	DefaultUpdateDownloadLinkExpire  = 24 * time.Hour
	DefaultImageGenerationLinkExpire = 7 * 24 * time.Hour
	PerPageInventoryDevices          = 512
	InventoryGroupScope              = "system"
	InventoryIdentityScope           = "identity"
	InventoryGroupAttributeName      = "group"
	InventoryStatusAttributeName     = "status"
	InventoryStatusAccepted          = "accepted"
)

// maximum image size is 10G
// NOTE: If this size increase, the buffer size in s3 client must be increased.
const MaxImageSize = 1024 * 1024 * 1024 * 10

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
	ErrModelImageInActiveDeployment     = errors.New(
		"Image is used in active deployment and cannot be removed",
	)
	ErrModelImageUsedInAnyDeployment = errors.New("Image has already been used in deployment")
	ErrModelParsingArtifactFailed    = errors.New("Cannot parse artifact file")

	ErrMsgArtifactConflict = "An artifact with the same name has conflicting dependencies"

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
	ErrNoDevices               = errors.New("No devices for the deployment")
	ErrDuplicateDeployment     = errors.New("Deployment with given ID already exists")
	ErrInvalidDeploymentID     = errors.New("Deployment ID must be a valid UUID")
)

//deployments

//go:generate ../utils/mockgen.sh
type App interface {
	HealthCheck(ctx context.Context) error
	// limits
	GetLimit(ctx context.Context, name string) (*model.Limit, error)
	ProvisionTenant(ctx context.Context, tenant_id string) error

	// Storage Settings
	GetStorageSettings(ctx context.Context) (*model.StorageSettings, error)
	SetStorageSettings(ctx context.Context, storageSettings *model.StorageSettings) error

	// images
	ListImages(
		ctx context.Context,
		filters *model.ReleaseOrImageFilter,
	) ([]*model.Image, int, error)
	DownloadLink(ctx context.Context, imageID string,
		expire time.Duration) (*model.Link, error)
	GetImage(ctx context.Context, id string) (*model.Image, error)
	DeleteImage(ctx context.Context, imageID string) error
	CreateImage(ctx context.Context,
		multipartUploadMsg *model.MultipartUploadMsg) (string, error)
	GenerateImage(ctx context.Context,
		multipartUploadMsg *model.MultipartGenerateImageMsg) (string, error)
	GenerateConfigurationImage(
		ctx context.Context,
		deviceType string,
		deploymentID string,
	) (io.Reader, error)
	EditImage(ctx context.Context, id string,
		constructorData *model.ImageMeta) (bool, error)

	// deployments
	CreateDeployment(ctx context.Context,
		constructor *model.DeploymentConstructor) (string, error)
	GetDeployment(ctx context.Context, deploymentID string) (*model.Deployment, error)
	IsDeploymentFinished(ctx context.Context, deploymentID string) (bool, error)
	AbortDeployment(ctx context.Context, deploymentID string) error
	GetDeploymentStats(ctx context.Context, deploymentID string) (model.Stats, error)
	GetDeploymentForDeviceWithCurrent(ctx context.Context, deviceID string,
		current *model.InstalledDeviceDeployment) (*model.DeploymentInstructions, error)
	HasDeploymentForDevice(ctx context.Context, deploymentID string,
		deviceID string) (bool, error)
	UpdateDeviceDeploymentStatus(ctx context.Context, deploymentID string,
		deviceID string, state model.DeviceDeploymentState) error
	GetDeviceStatusesForDeployment(ctx context.Context,
		deploymentID string) ([]model.DeviceDeployment, error)
	GetDevicesListForDeployment(ctx context.Context,
		query store.ListQuery) ([]model.DeviceDeployment, int, error)
	LookupDeployment(ctx context.Context,
		query model.Query) ([]*model.Deployment, int64, error)
	SaveDeviceDeploymentLog(ctx context.Context, deviceID string,
		deploymentID string, logs []model.LogMessage) error
	GetDeviceDeploymentLog(ctx context.Context,
		deviceID, deploymentID string) (*model.DeploymentLog, error)
	DecommissionDevice(ctx context.Context, deviceID string) error
	CreateDeviceConfigurationDeployment(
		ctx context.Context, constructor *model.ConfigurationDeploymentConstructor,
		deviceID, deploymentID string) (string, error)
}

type Deployments struct {
	db               store.DataStore
	fileStorage      s3.FileStorage
	imageContentType string
	workflowsClient  workflows.Client
	inventoryClient  inventory.Client
	reportingClient  reporting.Client
}

func NewDeployments(
	storage store.DataStore,
	fileStorage s3.FileStorage,
	imageContentType string,
) *Deployments {
	return &Deployments{
		db:               storage,
		fileStorage:      fileStorage,
		imageContentType: imageContentType,
		workflowsClient:  workflows.NewClient(),
		inventoryClient:  inventory.NewClient(),
	}
}

func (d *Deployments) SetWorkflowsClient(workflowsClient workflows.Client) {
	d.workflowsClient = workflowsClient
}

func (d *Deployments) SetInventoryClient(inventoryClient inventory.Client) {
	d.inventoryClient = inventoryClient
}

func (d *Deployments) HealthCheck(ctx context.Context) error {
	err := d.db.Ping(ctx)
	if err != nil {
		return errors.Wrap(err, "error reaching MongoDB")
	}
	fileStorage, err := d.getFileStorage(ctx)
	if err != nil {
		return err
	}
	_, err = fileStorage.ListBuckets(ctx)
	if err != nil {
		return errors.Wrap(
			err,
			"error reaching artifact storage service",
		)
	}

	err = d.workflowsClient.CheckHealth(ctx)
	if err != nil {
		return errors.Wrap(err, "Workflows service unhealthy")
	}

	err = d.inventoryClient.CheckHealth(ctx)
	if err != nil {
		return errors.Wrap(err, "Inventory service unhealthy")
	}

	if d.reportingClient != nil {
		err = d.reportingClient.CheckHealth(ctx)
		if err != nil {
			return errors.Wrap(err, "Reporting service unhealthy")
		}
	}
	return nil
}

func (d *Deployments) getFileStorage(ctx context.Context) (s3.FileStorage, error) {
	settings, err := d.db.GetStorageSettings(ctx)
	if err != nil {
		return nil, err
	} else if settings != nil && settings.Bucket != "" {
		region := config.Config.GetString(dconfig.SettingAwsS3Region)
		key := config.Config.GetString(dconfig.SettingAwsAuthKeyId)
		secret := config.Config.GetString(dconfig.SettingAwsAuthSecret)
		uri := config.Config.GetString(dconfig.SettingAwsURI)
		token := config.Config.GetString(dconfig.SettingAwsAuthToken)
		tagArtifact := config.Config.GetBool(dconfig.SettingsAwsTagArtifact)
		if settings.Region != "" {
			region = settings.Region
		}
		if settings.Key != "" {
			key = settings.Key
		}
		if settings.Secret != "" {
			secret = settings.Secret
		}
		if settings.Uri != "" {
			uri = settings.Uri
		}
		if settings.Token != "" {
			token = settings.Token
		}

		return s3.NewSimpleStorageServiceStatic(
			settings.Bucket,
			key,
			secret,
			region,
			token,
			uri,
			tagArtifact,
			settings.ForcePathStyle,
			settings.UseAccelerate,
		)
	}

	return d.fileStorage, nil
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

	if multipartUploadMsg.ArtifactSize > MaxImageSize {
		return "", ErrModelArtifactFileTooLarge
	}

	artifactID, err := d.handleArtifact(ctx, multipartUploadMsg)
	// try to remove artifact file from file storage on error
	if err != nil {
		fileStorage, err := d.getFileStorage(ctx)
		if err != nil {
			return "", err
		}
		if cleanupErr := fileStorage.Delete(ctx,
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
	lr := &utils.LimitedReader{
		R:          multipartUploadMsg.ArtifactReader,
		N:          multipartUploadMsg.ArtifactSize + 1,
		LimitError: ErrModelArtifactFileTooLarge,
	}
	tee := io.TeeReader(lr, pW)

	uid, err := uuid.FromString(multipartUploadMsg.ArtifactID)
	if err != nil {
		uid = uuid.NewV4()
	}
	artifactID := uid.String()

	ch := make(chan error)
	// create goroutine for artifact upload
	//
	// reading from the pipe (which is done in UploadArtifact method) is a blocking operation
	// and cannot be done in the same goroutine as writing to the pipe
	//
	// uploading and parsing artifact in the same process will cause in a deadlock!
	//nolint:errcheck
	go func() (err error) {
		defer func() { ch <- err }()
		fileStorage, err := d.getFileStorage(ctx)
		if err != nil {
			return err
		}
		err = fileStorage.UploadArtifact(
			ctx, artifactID, pR, ArtifactContentType,
		)
		if err != nil {
			pR.CloseWithError(err)
		}
		return err
	}()

	// parse artifact
	// artifact library reads all the data from the given reader
	metaArtifactConstructor, err := getMetaFromArchive(&tee)
	if err != nil {
		pW.Close()
		<-ch
		return artifactID, errors.Wrap(ErrModelParsingArtifactFailed, err.Error())
	}

	// read the rest of the data,
	// just in case the artifact library did not read all the data from the reader
	_, err = io.Copy(ioutil.Discard, tee)
	if err != nil {
		// CloseWithError will cause the reading end to abort upload.
		_ = pW.CloseWithError(err)
		<-ch
		return artifactID, err
	}

	// close the pipe
	pW.Close()

	// Assign artifact size to the actual uploaded size
	// NOTE: the limited reader is capped at one over the size limit.
	multipartUploadMsg.
		ArtifactSize = multipartUploadMsg.ArtifactSize - (lr.N - 1)

	// collect output from the goroutine
	if uploadResponseErr := <-ch; uploadResponseErr != nil {
		return artifactID, uploadResponseErr
	}

	// validate artifact metadata
	if err = metaArtifactConstructor.Validate(); err != nil {
		return artifactID, ErrModelInvalidMetadata
	}

	image := model.NewImage(
		artifactID,
		multipartUploadMsg.MetaConstructor,
		metaArtifactConstructor,
		multipartUploadMsg.ArtifactSize,
	)

	// save image structure in the system
	if err = d.db.InsertImage(ctx, image); err != nil {
		if idxErr, ok := err.(*model.ConflictError); ok {
			return artifactID, idxErr
		}
		return artifactID, errors.Wrap(err, "Fail to store the metadata")
	}

	return artifactID, nil
}

// GenerateImage parses raw data and uploads it to the file storage - in parallel,
// creates image structure in the system, and starts the workflow to generate the
// artifact from them.
// Returns image ID and nil on success.
func (d *Deployments) GenerateImage(ctx context.Context,
	multipartGenerateImageMsg *model.MultipartGenerateImageMsg) (string, error) {

	if multipartGenerateImageMsg == nil {
		return "", ErrModelMultipartUploadMsgMalformed
	}

	imgID, err := d.handleRawFile(ctx, multipartGenerateImageMsg)
	if err != nil {
		return "", err
	}

	multipartGenerateImageMsg.ArtifactID = imgID
	if id := identity.FromContext(ctx); id != nil && len(id.Tenant) > 0 {
		multipartGenerateImageMsg.TenantID = id.Tenant
	}

	fileStorage, err := d.getFileStorage(ctx)
	if err != nil {
		return "", err
	}
	link, err := fileStorage.GetRequest(
		ctx,
		imgID,
		DefaultImageGenerationLinkExpire,
		ArtifactContentType,
		"",
	)
	if err != nil {
		return "", err
	}
	multipartGenerateImageMsg.GetArtifactURI = link.Uri

	link, err = fileStorage.DeleteRequest(ctx, imgID, DefaultImageGenerationLinkExpire)
	if err != nil {
		return "", err
	}
	multipartGenerateImageMsg.DeleteArtifactURI = link.Uri

	err = d.workflowsClient.StartGenerateArtifact(ctx, multipartGenerateImageMsg)
	if err != nil {
		if cleanupErr := fileStorage.Delete(ctx, imgID); cleanupErr != nil {
			return "", errors.Wrap(err, cleanupErr.Error())
		}
		return "", err
	}

	return imgID, err
}

func (d *Deployments) GenerateConfigurationImage(
	ctx context.Context,
	deviceType string,
	deploymentID string,
) (io.Reader, error) {
	var buf bytes.Buffer
	dpl, err := d.db.FindDeploymentByID(ctx, deploymentID)
	if err != nil {
		return nil, err
	} else if dpl == nil {
		return nil, ErrModelDeploymentNotFound
	}
	var metaData map[string]interface{}
	err = json.Unmarshal(dpl.Configuration, &metaData)
	if err != nil {
		return nil, errors.Wrapf(err, "malformed configuration in deployment")
	}

	artieWriter := awriter.NewWriter(&buf, artifact.NewCompressorNone())
	module := handlers.NewModuleImage(ArtifactConfigureType)
	err = artieWriter.WriteArtifact(&awriter.WriteArtifactArgs{
		Format:  "mender",
		Version: 3,
		Devices: []string{deviceType},
		Name:    dpl.ArtifactName,
		Updates: &awriter.Updates{Updates: []handlers.Composer{module}},
		Depends: &artifact.ArtifactDepends{
			CompatibleDevices: []string{deviceType},
		},
		Provides: &artifact.ArtifactProvides{
			ArtifactName: dpl.ArtifactName,
		},
		MetaData: metaData,
		TypeInfoV3: &artifact.TypeInfoV3{
			Type: ArtifactConfigureType,
			ArtifactProvides: artifact.TypeInfoProvides{
				ArtifactConfigureProvides: dpl.ArtifactName,
			},
			ArtifactDepends:        artifact.TypeInfoDepends{},
			ClearsArtifactProvides: []string{ArtifactConfigureProvidesCleared},
		},
	})

	return &buf, err
}

// handleRawFile parses raw data, uploads it to the file storage,
// and starts the workflow to generate the artifact.
// Returns image ID, artifact file ID and nil on success.
func (d *Deployments) handleRawFile(ctx context.Context,
	multipartMsg *model.MultipartGenerateImageMsg) (string, error) {

	uid := uuid.NewV4()
	artifactID := uid.String()

	// check if artifact is unique
	// artifact is considered to be unique if there is no artifact with the same name
	// and supporting the same platform in the system
	isArtifactUnique, err := d.db.IsArtifactUnique(ctx,
		multipartMsg.Name,
		multipartMsg.DeviceTypesCompatible,
	)
	if err != nil {
		return "", errors.Wrap(err, "Fail to check if artifact is unique")
	}
	if !isArtifactUnique {
		return "", ErrModelArtifactNotUnique
	}

	file := &utils.LimitedReader{
		R:          multipartMsg.FileReader,
		N:          MaxImageSize + 1,
		LimitError: ErrModelArtifactFileTooLarge,
	}

	fileStorage, err := d.getFileStorage(ctx)
	if err != nil {
		return "", err
	}
	err = fileStorage.UploadArtifact(
		ctx, artifactID, file, ArtifactContentType,
	)
	if err != nil {
		return "", err
	}

	return artifactID, nil
}

// GetImage allows to fetch image object with specified id
// Nil if not found
func (d *Deployments) GetImage(ctx context.Context, id string) (*model.Image, error) {

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
// Noop for not existing images
// Allowed to remove image only if image is not scheduled or in progress for an updates - then image
// file is needed
// In case of already finished updates only image file is not needed, metadata is attached directly
// to device deployment therefore we still have some information about image that have been used
// (but not the file)
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
	fileStorage, err := d.getFileStorage(ctx)
	if err != nil {
		return err
	}
	if err := fileStorage.Delete(ctx, imageID); err != nil {
		return errors.Wrap(err, "Deleting image file")
	}

	// Delete metadata
	if err := d.db.DeleteImage(ctx, imageID); err != nil {
		return errors.Wrap(err, "Deleting image metadata")
	}

	return nil
}

// ListImages according to specified filers.
func (d *Deployments) ListImages(
	ctx context.Context,
	filters *model.ReleaseOrImageFilter,
) ([]*model.Image, int, error) {
	imageList, count, err := d.db.ListImages(ctx, filters)
	if err != nil {
		return nil, 0, errors.Wrap(err, "Searching for image metadata")
	}

	if imageList == nil {
		return make([]*model.Image, 0), 0, nil
	}

	return imageList, count, nil
}

// EditObject allows editing only if image have not been used yet in any deployment.
func (d *Deployments) EditImage(ctx context.Context, imageID string,
	constructor *model.ImageMeta) (bool, error) {

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
	foundImage.ImageMeta = constructor

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

	image, err := d.GetImage(ctx, imageID)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image with specified ID")
	}

	if image == nil {
		return nil, nil
	}

	fileStorage, err := d.getFileStorage(ctx)
	if err != nil {
		return nil, err
	}

	found, err := fileStorage.Exists(ctx, imageID)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for image file")
	}

	if !found {
		return nil, nil
	}

	fileName := image.ArtifactMeta.Name + ".mender"
	link, err := fileStorage.GetRequest(ctx, imageID,
		expire, ArtifactContentType, fileName)
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

func getMetaFromArchive(r *io.Reader) (*model.ArtifactMeta, error) {
	metaArtifact := model.NewArtifactMeta()

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
	if metaArtifact.Info.Version == 3 {
		metaArtifact.Depends, err = aReader.MergeArtifactDepends()
		if err != nil {
			return nil, errors.Wrap(err,
				"error parsing version 3 artifact")
		}

		metaArtifact.Provides, err = aReader.MergeArtifactProvides()
		if err != nil {
			return nil, errors.Wrap(err,
				"error parsing version 3 artifact")
		}

		metaArtifact.ClearsProvides = aReader.MergeArtifactClearsProvides()
	}

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

func getArtifactIDs(artifacts []*model.Image) []string {
	artifactIDs := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactIDs = append(artifactIDs, artifact.Id)
	}
	return artifactIDs
}

// deployments
func inventoryDevicesToDevicesIds(devices []model.InvDevice) []string {
	ids := make([]string, len(devices))
	for i, d := range devices {
		ids[i] = d.ID
	}

	return ids
}

// updateDeploymentConstructor fills devices list with device ids
func (d *Deployments) updateDeploymentConstructor(ctx context.Context,
	constructor *model.DeploymentConstructor) (*model.DeploymentConstructor, error) {
	l := log.FromContext(ctx)

	id := identity.FromContext(ctx)
	if id == nil {
		l.Error("identity not present in the context")
		return nil, ErrModelInternal
	}
	searchParams := model.SearchParams{
		Page:    1,
		PerPage: PerPageInventoryDevices,
		Filters: []model.FilterPredicate{
			{
				Scope:     InventoryIdentityScope,
				Attribute: InventoryStatusAttributeName,
				Type:      "$eq",
				Value:     InventoryStatusAccepted,
			},
		},
	}
	if len(constructor.Group) > 0 {
		searchParams.Filters = append(
			searchParams.Filters,
			model.FilterPredicate{
				Scope:     InventoryGroupScope,
				Attribute: InventoryGroupAttributeName,
				Type:      "$eq",
				Value:     constructor.Group,
			})
	}

	for {
		devices, count, err := d.search(ctx, id.Tenant, searchParams)
		if err != nil {
			l.Errorf("error searching for devices")
			return nil, ErrModelInternal
		}
		if count < 1 {
			l.Errorf("no devices found")
			return nil, ErrNoDevices
		}
		if len(devices) < 1 {
			break
		}
		constructor.Devices = append(constructor.Devices, inventoryDevicesToDevicesIds(devices)...)
		if len(constructor.Devices) == count {
			break
		}
		searchParams.Page++
	}

	return constructor, nil
}

// CreateDeviceConfigurationDeployment creates new configuration deployment for the device.
func (d *Deployments) CreateDeviceConfigurationDeployment(
	ctx context.Context, constructor *model.ConfigurationDeploymentConstructor,
	deviceID, deploymentID string) (string, error) {

	if constructor == nil {
		return "", ErrModelMissingInput
	}

	deployment, err := model.NewDeploymentFromConfigurationDeploymentConstructor(
		constructor,
		deploymentID,
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to create deployment")
	}

	deployment.DeviceList = []string{deviceID}
	deployment.MaxDevices = 1
	deployment.Configuration = []byte(constructor.Configuration)
	deployment.Type = model.DeploymentTypeConfiguration

	groups, err := d.getDeploymentGroups(ctx, []string{deviceID})
	if err != nil {
		return "", err
	}
	deployment.Groups = groups

	if err := d.db.InsertDeployment(ctx, deployment); err != nil {
		if strings.Contains(err.Error(), "duplicate key error") {
			return "", ErrDuplicateDeployment
		}
		if strings.Contains(err.Error(), "id: must be a valid UUID") {
			return "", ErrInvalidDeploymentID
		}
		return "", errors.Wrap(err, "Storing deployment data")
	}

	return deployment.Id, nil
}

// CreateDeployment precomputes new deployment and schedules it for devices.
func (d *Deployments) CreateDeployment(ctx context.Context,
	constructor *model.DeploymentConstructor) (string, error) {

	var err error

	if constructor == nil {
		return "", ErrModelMissingInput
	}

	if err := constructor.Validate(); err != nil {
		return "", errors.Wrap(err, "Validating deployment")
	}

	if len(constructor.Group) > 0 || constructor.AllDevices {
		constructor, err = d.updateDeploymentConstructor(ctx, constructor)
		if err != nil {
			return "", err
		}
	}

	deployment, err := model.NewDeploymentFromConstructor(constructor)
	if err != nil {
		return "", errors.Wrap(err, "failed to create deployment")
	}

	// Assign artifacts to the deployment.
	// Only artifacts present in the system at the moment of deployment creation
	// will be part of this deployment.
	artifacts, err := d.db.ImagesByName(ctx, deployment.ArtifactName)
	if err != nil {
		return "", errors.Wrap(err, "Finding artifact with given name")
	}

	if len(artifacts) == 0 {
		return "", ErrNoArtifact
	}

	deployment.Artifacts = getArtifactIDs(artifacts)
	deployment.DeviceList = constructor.Devices
	deployment.MaxDevices = len(constructor.Devices)
	deployment.Type = model.DeploymentTypeSoftware
	if len(constructor.Group) > 0 {
		deployment.Groups = []string{constructor.Group}
	}

	// single device deployment case
	if len(deployment.Groups) == 0 && len(constructor.Devices) == 1 {
		groups, err := d.getDeploymentGroups(ctx, constructor.Devices)
		if err != nil {
			return "", err
		}
		deployment.Groups = groups
	}

	if err := d.db.InsertDeployment(ctx, deployment); err != nil {
		return "", errors.Wrap(err, "Storing deployment data")
	}

	return deployment.Id, nil
}

func (d *Deployments) getDeploymentGroups(
	ctx context.Context,
	devices []string,
) ([]string, error) {
	id := identity.FromContext(ctx)

	//only for single device deployment case
	if len(devices) != 1 {
		return nil, nil
	}

	if id == nil {
		id = &identity.Identity{}
	}

	groups, err := d.inventoryClient.GetDeviceGroups(ctx, id.Tenant, devices[0])
	if err != nil {
		return nil, err
	}
	return groups, nil
}

// IsDeploymentFinished checks if there is unfinished deployment with given ID
func (d *Deployments) IsDeploymentFinished(
	ctx context.Context,
	deploymentID string,
) (bool, error) {
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

	if err := d.setDeploymentDeviceCountIfUnset(ctx, deployment); err != nil {
		return nil, err
	}

	return deployment, nil
}

// ImageUsedInActiveDeployment checks if specified image is in use by deployments Image is
// considered to be in use if it's participating in at lest one non success/error deployment.
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

// assignArtifact assigns artifact to the device deployment
func (d *Deployments) assignArtifact(
	ctx context.Context,
	deployment *model.Deployment,
	deviceDeployment *model.DeviceDeployment,
	installed *model.InstalledDeviceDeployment) error {

	// Assign artifact to the device deployment.
	var artifact *model.Image
	var err error

	if err = installed.Validate(); err != nil {
		return err
	}

	if deviceDeployment.DeploymentId == "" || deviceDeployment.DeviceId == "" {
		return ErrModelInternal
	}

	// Clear device deployment image
	// New artifact will be selected for the device deployment
	deviceDeployment.Image = nil

	// First case is for backward compatibility.
	// It is possible that there is old deployment structure in the system.
	// In such case we need to select artifact using name and device type.
	if deployment.Artifacts == nil || len(deployment.Artifacts) == 0 {
		artifact, err = d.db.ImageByNameAndDeviceType(
			ctx,
			installed.ArtifactName,
			installed.DeviceType,
		)
		if err != nil {
			return errors.Wrap(err, "assigning artifact to device deployment")
		}
	} else {
		// Select artifact for the device deployment from artifacts assigned to the deployment.
		artifact, err = d.db.ImageByIdsAndDeviceType(
			ctx,
			deployment.Artifacts,
			installed.DeviceType,
		)
		if err != nil {
			return errors.Wrap(err, "assigning artifact to device deployment")
		}
	}

	// If not having appropriate image, set noartifact status
	if artifact == nil {
		if err := d.UpdateDeviceDeploymentStatus(ctx, deviceDeployment.DeploymentId,
			deviceDeployment.DeviceId,
			model.DeviceDeploymentState{
				Status: model.DeviceDeploymentStatusNoArtifact,
			}); err != nil {
			return errors.Wrap(err, "Failed to update deployment status")
		}
		return nil
	}

	if err := d.db.AssignArtifact(
		ctx, deviceDeployment.DeviceId, deviceDeployment.DeploymentId, artifact); err != nil {
		return errors.Wrap(err, "Assigning artifact to the device deployment")
	}

	deviceDeployment.Image = artifact
	deviceDeployment.DeviceType = installed.DeviceType

	return nil
}

// Retrieves the model.Deployment and model.DeviceDeployment structures
// for the device. Upon error, nil is returned for both deployment structures.
func (d *Deployments) getDeploymentForDevice(ctx context.Context,
	deviceID string) (*model.Deployment, *model.DeviceDeployment, error) {

	// Retrieve device deployment
	deviceDeployment, err := d.db.FindOldestDeploymentForDeviceIDWithStatuses(
		ctx,
		deviceID,
		model.ActiveDeploymentStatuses()...)

	if err != nil {
		return nil, nil, errors.Wrap(err,
			"Searching for oldest active deployment for the device")
	} else if deviceDeployment == nil {
		return d.getNewDeploymentForDevice(ctx, deviceID)
	}

	deployment, err := d.db.FindDeploymentByID(ctx, deviceDeployment.DeploymentId)
	if err != nil {
		return nil, nil, errors.Wrap(err, "checking deployment id")
	}
	if deployment == nil {
		return nil, nil, errors.New("No deployment corresponding to device deployment")
	}

	return deployment, deviceDeployment, nil
}

// getNewDeploymentForDevice returns deployment object and creates and returns
// new device deployment for the device;
//
// we are interested only in the deployments that are newer than the latest
// deployment applied by the device;
// this way we guarantee that the device will not receive deployment
// that is older than the one installed on the device;
func (d *Deployments) getNewDeploymentForDevice(ctx context.Context,
	deviceID string) (*model.Deployment, *model.DeviceDeployment, error) {

	var lastDeployment *time.Time
	//get latest device deployment for the device;
	deviceDeployment, err := d.db.FindLatestDeploymentForDeviceIDWithStatuses(
		ctx,
		deviceID,
		model.InactiveDeploymentStatuses()...)
	if err != nil {
		return nil, nil, errors.Wrap(err,
			"Searching for latest active deployment for the device")
	} else if deviceDeployment == nil {
		lastDeployment = &time.Time{}
	} else {
		lastDeployment = deviceDeployment.Created
	}

	//get deployments newer then last device deployment
	//iterate over deployments and check if the device is part of the deployment or not
	for skip := 0; true; skip += 100 {
		deployments, err := d.db.FindNewerActiveDeployments(ctx, lastDeployment, skip, 100)
		if err != nil {
			return nil, nil, errors.Wrap(err,
				"Failed to search for newer active deployments")
		}
		if len(deployments) == 0 {
			return nil, nil, nil
		}

		for _, deployment := range deployments {
			ok, err := d.isDevicePartOfDeployment(ctx, deviceID, deployment)
			if err != nil {
				return nil, nil, err
			}
			if ok {
				deviceDeployment, err := d.createDeviceDeploymentWithStatus(ctx,
					deviceID, deployment, model.DeviceDeploymentStatusPending)
				if err != nil {
					return nil, nil, err
				}
				return deployment, deviceDeployment, nil
			}
		}
	}

	return nil, nil, nil
}

func (d *Deployments) createDeviceDeploymentWithStatus(
	ctx context.Context, deviceID string,
	deployment *model.Deployment, status model.DeviceDeploymentStatus,
) (*model.DeviceDeployment, error) {
	deviceDeployment := model.NewDeviceDeployment(deviceID, deployment.Id)

	deviceDeployment.Status = status
	deviceDeployment.Created = deployment.Created

	if err := d.setDeploymentDeviceCountIfUnset(ctx, deployment); err != nil {
		return nil, err
	}

	if err := d.db.InsertDeviceDeployment(ctx, deviceDeployment); err != nil {
		return nil, err
	}

	// after inserting new device deployment update deployment stats
	// in the database and locally, and update deployment status
	if err := d.db.UpdateStatsInc(
		ctx, deployment.Id,
		model.DeviceDeploymentStatusNull, status,
	); err != nil {
		return nil, err
	}

	deployment.Stats.Inc(status)

	err := d.recalcDeploymentStatus(ctx, deployment)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update deployment status")
	}

	return deviceDeployment, nil
}

func (d *Deployments) isDevicePartOfDeployment(
	ctx context.Context,
	deviceID string,
	deployment *model.Deployment,
) (bool, error) {
	for _, id := range deployment.DeviceList {
		if id == deviceID {
			return true, nil
		}
	}
	return false, nil
}

// GetDeploymentForDeviceWithCurrent returns deployment for the device
func (d *Deployments) GetDeploymentForDeviceWithCurrent(ctx context.Context, deviceID string,
	installed *model.InstalledDeviceDeployment) (*model.DeploymentInstructions, error) {

	deployment, deviceDeployment, err := d.getDeploymentForDevice(ctx, deviceID)
	if err != nil {
		return nil, ErrModelInternal
	}

	if deployment == nil {
		return nil, nil
	}

	if installed.ArtifactName != "" && deployment.ArtifactName == installed.ArtifactName {
		// pretend there is no deployment for this device, but update
		// its status to already installed first

		if err := d.UpdateDeviceDeploymentStatus(ctx, deviceDeployment.DeploymentId, deviceID,
			model.DeviceDeploymentState{
				Status: model.DeviceDeploymentStatusAlreadyInst,
			}); err != nil {

			return nil, errors.Wrap(err, "Failed to update deployment status")
		}

		return nil, nil
	}
	if deployment.Type == model.DeploymentTypeConfiguration {
		// There's nothing more we need to do, the link must be filled
		// in by the API layer.
		return &model.DeploymentInstructions{
			ID: deployment.Id,
			Artifact: model.ArtifactDeploymentInstructions{
				ArtifactName:          deployment.ArtifactName,
				DeviceTypesCompatible: []string{installed.DeviceType},
			},
			Type: model.DeploymentTypeConfiguration,
		}, nil
	}

	// assign artifact only if the artifact was not assigned previously or the device type has
	// changed
	if deviceDeployment.Image == nil ||
		len(deviceDeployment.DeviceType) == 0 ||
		deviceDeployment.DeviceType != installed.DeviceType {

		if err := d.assignArtifact(ctx, deployment, deviceDeployment, installed); err != nil {
			return nil, err
		}
	}

	if deviceDeployment.Image == nil {
		return nil, nil
	}

	fileStorage, err := d.getFileStorage(ctx)
	if err != nil {
		return nil, err
	}

	link, err := fileStorage.GetRequest(ctx, deviceDeployment.Image.Id,
		DefaultUpdateDownloadLinkExpire, d.imageContentType, "")
	if err != nil {
		return nil, errors.Wrap(err, "Generating download link for the device")
	}

	instructions := &model.DeploymentInstructions{
		ID: deviceDeployment.DeploymentId,
		Artifact: model.ArtifactDeploymentInstructions{
			ArtifactName: deviceDeployment.Image.
				ArtifactMeta.Name,
			Source: *link,
			DeviceTypesCompatible: deviceDeployment.Image.
				ArtifactMeta.DeviceTypesCompatible,
		},
	}

	return instructions, nil
}

// UpdateDeviceDeploymentStatus will update the deployment status for device of
// ID `deviceID`. Returns nil if update was successful.
func (d *Deployments) UpdateDeviceDeploymentStatus(ctx context.Context, deploymentID string,
	deviceID string, ddState model.DeviceDeploymentState) error {

	l := log.FromContext(ctx)

	l.Infof("New status: %s for device %s deployment: %v", ddState.Status, deviceID, deploymentID)

	var finishTime *time.Time = nil
	if model.IsDeviceDeploymentStatusFinished(ddState.Status) {
		now := time.Now()
		finishTime = &now
	}

	dd, err := d.db.GetDeviceDeployment(ctx, deploymentID, deviceID)
	if err != nil {
		return err
	}

	currentStatus := dd.Status

	if currentStatus == model.DeviceDeploymentStatusAborted {
		return ErrDeploymentAborted
	}

	if currentStatus == model.DeviceDeploymentStatusDecommissioned {
		return ErrDeviceDecommissioned
	}

	// nothing to do
	if ddState.Status == currentStatus {
		return nil
	}

	// update finish time
	ddState.FinishTime = finishTime

	old, err := d.db.UpdateDeviceDeploymentStatus(ctx,
		deviceID, deploymentID, ddState)
	if err != nil {
		return err
	}

	if err = d.db.UpdateStatsInc(ctx, deploymentID, old, ddState.Status); err != nil {
		return err
	}

	// fetch deployment stats and update deployment status
	deployment, err := d.db.FindDeploymentByID(ctx, deploymentID)
	if err != nil {
		return errors.Wrap(err, "failed when searching for deployment")
	}

	err = d.recalcDeploymentStatus(ctx, deployment)
	if err != nil {
		return errors.Wrap(err, "failed to update deployment status")
	}

	return nil
}

// recalcDeploymentStatus inspects the deployment stats and
// recalculates and updates its status
// it should be used whenever deployment stats are touched
func (d *Deployments) recalcDeploymentStatus(ctx context.Context, dep *model.Deployment) error {
	status := dep.GetStatus()

	if err := d.db.SetDeploymentStatus(ctx, dep.Id, status, time.Now()); err != nil {
		return err
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

	return deployment.Stats, nil
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

func (d *Deployments) GetDevicesListForDeployment(ctx context.Context,
	query store.ListQuery) ([]model.DeviceDeployment, int, error) {

	deployment, err := d.db.FindDeploymentByID(ctx, query.DeploymentID)
	if err != nil {
		return nil, -1, ErrModelInternal
	}

	if deployment == nil {
		return nil, -1, ErrModelDeploymentNotFound
	}

	statuses, totalCount, err := d.db.GetDevicesListForDeployment(ctx, query)
	if err != nil {
		return nil, -1, ErrModelInternal
	}

	return statuses, totalCount, nil
}

func (d *Deployments) setDeploymentDeviceCountIfUnset(
	ctx context.Context,
	deployment *model.Deployment,
) error {
	if deployment.DeviceCount == nil {
		deviceCount, err := d.db.DeviceCountByDeployment(ctx, deployment.Id)
		if err != nil {
			return errors.Wrap(err, "counting device deployments")
		}
		err = d.db.SetDeploymentDeviceCount(ctx, deployment.Id, deviceCount)
		if err != nil {
			return errors.Wrap(err, "setting the device count for the deployment")
		}
		deployment.DeviceCount = &deviceCount
	}

	return nil
}

func (d *Deployments) LookupDeployment(ctx context.Context,
	query model.Query) ([]*model.Deployment, int64, error) {
	list, totalCount, err := d.db.Find(ctx, query)

	if err != nil {
		return nil, 0, errors.Wrap(err, "searching for deployments")
	}

	if list == nil {
		return make([]*model.Deployment, 0), 0, nil
	}

	for _, deployment := range list {
		if err := d.setDeploymentDeviceCountIfUnset(ctx, deployment); err != nil {
			return nil, 0, err
		}
	}

	return list, totalCount, nil
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

	// update statistics
	if err := d.db.UpdateStats(ctx, deploymentID, stats); err != nil {
		return errors.Wrap(err, "failed to update deployment stats")
	}

	// when aborting the deployment we need to set status directly instead of
	// using recalcDeploymentStatus method;
	// it is possible that the deployment does not have any device deployments yet;
	// in that case, all statistics are 0 and calculating status based on statistics
	// will not work - the calculated status will be "pending"
	if err := d.db.SetDeploymentStatus(ctx,
		deploymentID, model.DeploymentStatusFinished, time.Now()); err != nil {
		return errors.Wrap(err, "failed to update deployment status")
	}

	return nil
}

func (d *Deployments) DecommissionDevice(ctx context.Context, deviceId string) error {

	var lastDeployment *time.Time
	// Retrieve active device deployment for the device
	deviceDeployment, err := d.db.FindOldestDeploymentForDeviceIDWithStatuses(
		ctx,
		deviceId,
		model.ActiveDeploymentStatuses()...)
	if err != nil {
		return errors.Wrap(err, "Searching for active deployment for the device")
	} else if deviceDeployment != nil {
		now := time.Now()
		ddStatus := model.DeviceDeploymentState{
			Status:     model.DeviceDeploymentStatusDecommissioned,
			FinishTime: &now,
		}
		if err := d.UpdateDeviceDeploymentStatus(ctx, deviceDeployment.DeploymentId,
			deviceId, ddStatus); err != nil {
			return errors.Wrap(err, "updating device deployment status")
		}
		lastDeployment = deviceDeployment.Created
	} else {
		//get latest device deployment for the device;
		deviceDeployment, err := d.db.FindLatestDeploymentForDeviceIDWithStatuses(
			ctx,
			deviceId,
			model.InactiveDeploymentStatuses()...)
		if err != nil {
			return errors.Wrap(err, "Searching for latest active deployment for the device")
		} else if deviceDeployment == nil {
			lastDeployment = &time.Time{}
		} else {
			lastDeployment = deviceDeployment.Created
		}
	}

	// get deployments newer then last device deployment
	// iterate over deployments and check if the device is part of the deployment or not
	// if the device is part of the deployment create new, decommisioned device deployment
	for skip := 0; true; skip += 100 {
		deployments, err := d.db.FindNewerActiveDeployments(ctx, lastDeployment, skip, 100)
		if err != nil {
			return errors.Wrap(err, "Failed to search for newer active deployments")
		}
		if len(deployments) == 0 {
			break
		}
		for _, deployment := range deployments {
			ok, err := d.isDevicePartOfDeployment(ctx, deviceId, deployment)
			if err != nil {
				return err
			}
			if ok {
				_, err := d.createDeviceDeploymentWithStatus(ctx,
					deviceId, deployment, model.DeviceDeploymentStatusDecommissioned)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Storage settings
func (d *Deployments) GetStorageSettings(ctx context.Context) (*model.StorageSettings, error) {
	settings, err := d.db.GetStorageSettings(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Searching for settings failed")
	}

	return settings, nil
}

func (d *Deployments) SetStorageSettings(
	ctx context.Context,
	storageSettings *model.StorageSettings,
) error {
	if err := storageSettings.Validate(); err != nil {
		return errors.Wrap(err, "Invalid input data")
	}

	if err := d.db.SetStorageSettings(ctx, storageSettings); err != nil {
		return errors.Wrap(err, "Failed to save settings")
	}

	return nil
}

func (d *Deployments) WithReporting(c reporting.Client) *Deployments {
	d.reportingClient = c
	return d
}

func (d *Deployments) haveReporting() bool {
	return d.reportingClient != nil
}

func (d *Deployments) search(
	ctx context.Context,
	tid string,
	parms model.SearchParams,
) ([]model.InvDevice, int, error) {
	if d.haveReporting() {
		return d.reportingClient.Search(ctx, tid, parms)
	} else {
		return d.inventoryClient.Search(ctx, tid, parms)
	}
}

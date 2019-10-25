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
package mongo

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/mendersoftware/go-lib-micro/config"
	mstore "github.com/mendersoftware/go-lib-micro/store"
	"github.com/pkg/errors"

	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/model"
)

const (
	DatabaseName                   = "deployment_service"
	CollectionLimits               = "limits"
	CollectionImages               = "images"
	CollectionDeployments          = "deployments"
	CollectionDeviceDeploymentLogs = "devices.logs"
	CollectionDevices              = "devices"
)

// Indexes
const (
	IndexUniqeNameAndDeviceTypeStr           = "uniqueNameAndDeviceTypeIndex"
	IndexDeploymentArtifactNameStr           = "deploymentArtifactNameIndex"
	IndexDeploymentDeviceStatusesStr         = "deviceIdWithStatusByCreated"
	IndexDeploymentDeviceIdStatusStr         = "devicesIdWithStatus"
	IndexDeploymentDeviceDeploymentIdStr     = "devicesDeploymentId"
	IndexDeploymentStatusFinishedStr         = "deploymentStatusFinished"
	IndexDeploymentStatusPendingStr          = "deploymentStatusPending"
	IndexDeploymentCreatedStr                = "deploymentCreated"
	IndexDeploymentDeviceStatusRebootingStr  = "deploymentsDeviceStatusRebooting"
	IndexDeploymentDeviceStatusPendingStr    = "deploymentsDeviceStatusPending"
	IndexDeploymentDeviceStatusInstallingStr = "deploymentsDeviceStatusInstalling"
	IndexDeploymentDeviceStatusFinishedStr   = "deploymentsFinished"
)

var (
	StorageIndexes = []string{
		"$text:" + StorageKeyDeploymentName,
		"$text:" + StorageKeyDeploymentArtifactName,
	}
	StatusIndexes = []string{
		StorageKeyDeviceDeploymentDeviceId,
		StorageKeyDeviceDeploymentStatus,
		StorageKeyDeploymentStatsCreated,
	}
	DeviceIDStatusIndexes         = []string{"deviceID", "status"} //IndexDeploymentDeviceIdStatusStr
	DeploymentIdIndexes           = []string{"deploymentid"}       //IndexDeploymentDeviceDeploymentIdStr
	DeploymentStatusFinishedIndex = []string{
		"stats.downloading",
		"stats.installing",
		"stats.pending",
		"stats.rebooting",
		"-created",
	} //IndexDeploymentStatusFinishedStr
	DeploymentStatusPendingIndex = []string{
		"stats.aborted",
		"stats.already-installed",
		"stats.decommissioned",
		"stats.downloading",
		"stats.failure",
		"stats.installing",
		"stats.noartifact",
		"stats.rebooting",
		"stats.success",
		"-created",
	} //IndexDeploymentStatusPendingStr
	DeploymentCreatedIndex                = []string{"-created"}         //IndexDeploymentCreatedStr
	DeploymentDeviceStatusRebootingIndex  = []string{"stats.rebooting"}  //IndexDeploymentDeviceStatusRebootingStr
	DeploymentDeviceStatusPendingIndex    = []string{"stats.pending"}    //IndexDeploymentDeviceStatusPendingStr
	DeploymentDeviceStatusInstallingIndex = []string{"stats.installing"} //IndexDeploymentDeviceStatusInstallingStr
	DeploymentDeviceStatusFinishedIndex   = []string{"finished"}         //IndexDeploymentDeviceStatusFinishedStr
)

// Errors
var (
	ErrSoftwareImagesStorageInvalidID           = errors.New("Invalid id")
	ErrSoftwareImagesStorageInvalidArtifactName = errors.New("Invalid artifact name")
	ErrSoftwareImagesStorageInvalidName         = errors.New("Invalid name")
	ErrSoftwareImagesStorageInvalidDeviceType   = errors.New("Invalid device type")
	ErrSoftwareImagesStorageInvalidImage        = errors.New("Invalid image")

	ErrStorageInvalidDeviceDeployment = errors.New("Invalid device deployment")

	ErrDeploymentStorageInvalidDeployment = errors.New("Invalid deployment")
	ErrStorageInvalidID                   = errors.New("Invalid id")
	ErrStorageNotFound                    = errors.New("Not found")
	ErrDeploymentStorageInvalidQuery      = errors.New("Invalid query")
	ErrDeploymentStorageCannotExecQuery   = errors.New("Cannot execute query")
	ErrStorageInvalidInput                = errors.New("invalid input")

	ErrLimitNotFound = errors.New("limit not found")
)

// Database keys
const (
	// Need to be kept in sync with structure filed names
	StorageKeySoftwareImageDeviceTypes = "meta_artifact.device_types_compatible"
	StorageKeySoftwareImageName        = "meta_artifact.name"
	StorageKeySoftwareImageId          = "_id"

	StorageKeyDeviceDeploymentLogMessages = "messages"

	StorageKeyDeviceDeploymentAssignedImage   = "image"
	StorageKeyDeviceDeploymentAssignedImageId = StorageKeyDeviceDeploymentAssignedImage + "." + StorageKeySoftwareImageId
	StorageKeyDeviceDeploymentDeviceId        = "deviceid"
	StorageKeyDeviceDeploymentStatus          = "status"
	StorageKeyDeviceDeploymentSubState        = "substate"
	StorageKeyDeviceDeploymentDeploymentID    = "deploymentid"
	StorageKeyDeviceDeploymentFinished        = "finished"
	StorageKeyDeviceDeploymentIsLogAvailable  = "log"
	StorageKeyDeviceDeploymentArtifact        = "image"

	StorageKeyDeploymentName         = "deploymentconstructor.name"
	StorageKeyDeploymentArtifactName = "deploymentconstructor.artifactname"
	StorageKeyDeploymentStats        = "stats"
	StorageKeyDeploymentStatsCreated = "created"
	StorageKeyDeploymentFinished     = "finished"
	StorageKeyDeploymentArtifacts    = "artifacts"
)

type DataStoreMongo struct {
	session *mgo.Session
}

func NewDataStoreMongoWithSession(session *mgo.Session) *DataStoreMongo {
	return &DataStoreMongo{
		session: session,
	}
}

func NewMongoSession(c config.Reader) (*mgo.Session, error) {

	dialInfo, err := mgo.ParseURL(c.GetString(dconfig.SettingMongo))
	if err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// Set 10s timeout - same as set by Dial
	dialInfo.Timeout = 10 * time.Second

	username := c.GetString(dconfig.SettingDbUsername)
	if username != "" {
		dialInfo.Username = username
	}

	passward := c.GetString(dconfig.SettingDbPassword)
	if passward != "" {
		dialInfo.Password = passward
	}

	if c.GetBool(dconfig.SettingDbSSL) {
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {

			// Setup TLS
			tlsConfig := &tls.Config{}
			tlsConfig.InsecureSkipVerify = c.GetBool(dconfig.SettingDbSSLSkipVerify)

			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		}
	}

	masterSession, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// Validate connection
	if err := masterSession.Ping(); err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// force write ack with immediate journal file fsync
	masterSession.SetSafe(&mgo.Safe{
		W: 1,
		J: true,
	})

	return masterSession, nil
}

func (db *DataStoreMongo) GetReleases(ctx context.Context, filt *model.ReleaseFilter) ([]model.Release, error) {
	session := db.session.Copy()
	defer session.Close()

	match := db.matchFromFilt(filt)

	group := bson.M{
		"$group": bson.M{
			"_id": "$" + StorageKeySoftwareImageName,
			"name": bson.M{
				"$first": "$" + StorageKeySoftwareImageName,
			},
			"artifacts": bson.M{
				"$push": "$$ROOT",
			},
		},
	}

	sort := bson.M{
		"$sort": bson.M{
			"name": -1,
		},
	}

	var pipe []bson.M

	if match != nil {
		pipe = []bson.M{
			match,
			group,
			sort,
		}
	} else {
		pipe = []bson.M{
			group,
			sort,
		}
	}

	results := []model.Release{}

	err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Pipe(&pipe).All(&results)
	if err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return results, nil
}

func (db *DataStoreMongo) matchFromFilt(f *model.ReleaseFilter) bson.M {
	if f == nil {
		return nil
	}

	return bson.M{
		"$match": bson.M{
			StorageKeySoftwareImageName: f.Name,
		},
	}
}

// limits
//
func (db *DataStoreMongo) GetLimit(ctx context.Context, name string) (*model.Limit, error) {

	session := db.session.Copy()
	defer session.Close()

	var limit model.Limit
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionLimits).FindId(name).One(&limit); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, ErrLimitNotFound
		}
		return nil, err
	}

	return &limit, nil
}

func (db *DataStoreMongo) ProvisionTenant(ctx context.Context, tenantId string) error {
	session := db.session.Copy()
	defer session.Close()

	dbname := mstore.DbNameForTenant(tenantId, DbName)

	return MigrateSingle(ctx, dbname, DbVersion, session, true)
}

//images

// Ensure required indexes exists; create if not.
func (db *DataStoreMongo) ensureIndexing(ctx context.Context, session *mgo.Session) error {

	uniqueNameVersionIndex := mgo.Index{
		Key:    []string{StorageKeySoftwareImageName, StorageKeySoftwareImageDeviceTypes},
		Unique: true,
		Name:   IndexUniqeNameAndDeviceTypeStr,
		// Build index upfront - make sure this index is always on.
		Background: false,
	}

	return session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).EnsureIndex(uniqueNameVersionIndex)
}

// Exists checks if object with ID exists
func (db *DataStoreMongo) Exists(ctx context.Context, id string) (bool, error) {

	if govalidator.IsNull(id) {
		return false, ErrSoftwareImagesStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	var image *model.SoftwareImage
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).FindId(id).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Update proviced SoftwareImage
// Return false if not found
func (db *DataStoreMongo) Update(ctx context.Context,
	image *model.SoftwareImage) (bool, error) {

	if err := image.Validate(); err != nil {
		return false, err
	}

	session := db.session.Copy()
	defer session.Close()

	image.SetModified(time.Now())
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).UpdateId(image.Id, image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ImageByNameAndDeviceType finds image with speficied application name and targed device type
func (db *DataStoreMongo) ImageByNameAndDeviceType(ctx context.Context,
	name, deviceType string) (*model.SoftwareImage, error) {

	if govalidator.IsNull(name) {
		return nil, ErrSoftwareImagesStorageInvalidName

	}

	if govalidator.IsNull(deviceType) {
		return nil, ErrSoftwareImagesStorageInvalidDeviceType
	}

	// equal to device type & software version (application name + version)
	query := bson.M{
		StorageKeySoftwareImageDeviceTypes: deviceType,
		StorageKeySoftwareImageName:        name,
	}

	session := db.session.Copy()
	defer session.Close()

	// Both we lookup uniqe object, should be one or none.
	var image model.SoftwareImage
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Find(query).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return &image, nil
}

// ImageByIdsAndDeviceType finds image with id from ids and targed device type
func (db *DataStoreMongo) ImageByIdsAndDeviceType(ctx context.Context,
	ids []string, deviceType string) (*model.SoftwareImage, error) {

	if govalidator.IsNull(deviceType) {
		return nil, ErrSoftwareImagesStorageInvalidDeviceType
	}

	if len(ids) == 0 {
		return nil, ErrSoftwareImagesStorageInvalidID
	}

	query := bson.M{
		StorageKeySoftwareImageDeviceTypes: deviceType,
		StorageKeySoftwareImageId:          bson.M{"$in": ids},
	}

	session := db.session.Copy()
	defer session.Close()

	// Both we lookup uniqe object, should be one or none.
	var image model.SoftwareImage
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Find(query).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return &image, nil
}

// ImagesByName finds images with speficied artifact name
func (db *DataStoreMongo) ImagesByName(
	ctx context.Context, name string) ([]*model.SoftwareImage, error) {

	if govalidator.IsNull(name) {
		return nil, ErrSoftwareImagesStorageInvalidName

	}

	// equal to artifact name
	query := bson.M{
		StorageKeySoftwareImageName: name,
	}

	session := db.session.Copy()
	defer session.Close()

	// Both we lookup uniqe object, should be one or none.
	var images []*model.SoftwareImage
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Find(query).All(&images); err != nil {
		return nil, err
	}

	return images, nil
}

// Insert persists object
func (db *DataStoreMongo) InsertImage(ctx context.Context, image *model.SoftwareImage) error {

	if image == nil {
		return ErrSoftwareImagesStorageInvalidImage
	}

	if err := image.Validate(); err != nil {
		return err
	}

	session := db.session.Copy()
	defer session.Close()

	if err := db.ensureIndexing(ctx, session); err != nil {
		return err
	}

	return session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Insert(image)
}

// FindImageByID search storage for image with ID, returns nil if not found
func (db *DataStoreMongo) FindImageByID(ctx context.Context,
	id string) (*model.SoftwareImage, error) {

	if govalidator.IsNull(id) {
		return nil, ErrSoftwareImagesStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	var image *model.SoftwareImage
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).FindId(id).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return image, nil
}

// IsArtifactUnique checks if there is no artifact with the same artifactName
// supporting one of the device types from deviceTypesCompatible list.
// Returns true, nil if artifact is unique;
// false, nil if artifact is not unique;
// false, error in case of error.
func (db *DataStoreMongo) IsArtifactUnique(ctx context.Context,
	artifactName string, deviceTypesCompatible []string) (bool, error) {

	if govalidator.IsNull(artifactName) {
		return false, ErrSoftwareImagesStorageInvalidArtifactName
	}

	session := db.session.Copy()
	defer session.Close()

	query := bson.M{
		"$and": []bson.M{
			{
				StorageKeySoftwareImageName: artifactName,
			},
			{
				StorageKeySoftwareImageDeviceTypes: bson.M{"$in": deviceTypesCompatible},
			},
		},
	}

	var image *model.SoftwareImage
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Find(query).One(&image); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return true, nil
		}
		return false, err
	}

	return false, nil
}

// Delete image specified by ID
// Noop on if not found.
func (db *DataStoreMongo) DeleteImage(ctx context.Context, id string) error {

	if govalidator.IsNull(id) {
		return ErrSoftwareImagesStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).RemoveId(id); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil
		}
		return err
	}

	return nil
}

// FindAll lists all images
func (db *DataStoreMongo) FindAll(ctx context.Context) ([]*model.SoftwareImage, error) {

	session := db.session.Copy()
	defer session.Close()

	var images []*model.SoftwareImage
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionImages).Find(nil).All(&images); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return images, nil
		}
		return nil, err
	}

	return images, nil
}

//device deployemnt log

func (db *DataStoreMongo) SaveDeviceDeploymentLog(ctx context.Context,
	log model.DeploymentLog) error {

	if err := log.Validate(); err != nil {
		return err
	}

	session := db.session.Copy()
	defer session.Close()

	query := bson.M{
		StorageKeyDeviceDeploymentDeviceId:     log.DeviceID,
		StorageKeyDeviceDeploymentDeploymentID: log.DeploymentID,
	}

	// update log messages
	// if the deployment log is already present than messages will be overwritten
	update := bson.M{
		"$set": bson.M{
			StorageKeyDeviceDeploymentLogMessages: log.Messages,
		},
	}
	if _, err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeviceDeploymentLogs).Upsert(query, update); err != nil {
		return err
	}

	return nil
}

func (db *DataStoreMongo) GetDeviceDeploymentLog(ctx context.Context,
	deviceID, deploymentID string) (*model.DeploymentLog, error) {

	session := db.session.Copy()
	defer session.Close()

	query := bson.M{
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
	}

	var depl model.DeploymentLog
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeviceDeploymentLogs).Find(query).One(&depl); err != nil {
		if err == mgo.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &depl, nil
}

// device deployments

// InsertMany stores multiple device deployment objects.
// TODO: Handle error cleanup, multi insert is not atomic, loop into two-phase commits
func (db *DataStoreMongo) InsertMany(ctx context.Context,
	deployments ...*model.DeviceDeployment) error {

	if len(deployments) == 0 {
		return nil
	}

	// Writing to another interface list addresses golang gatcha interface{} == []interface{}
	var list []interface{}
	for _, deployment := range deployments {

		if deployment == nil {
			return ErrStorageInvalidDeviceDeployment
		}

		if err := deployment.Validate(); err != nil {
			return errors.Wrap(err, "Validating device deployment")
		}

		list = append(list, deployment)
	}

	session := db.session.Copy()
	defer session.Close()

	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Insert(list...); err != nil {
		return err
	}

	return nil
}

// ExistAssignedImageWithIDAndStatuses checks if image is used by deplyment with specified status.
func (db *DataStoreMongo) ExistAssignedImageWithIDAndStatuses(ctx context.Context,
	imageID string, statuses ...string) (bool, error) {

	// Verify ID formatting
	if govalidator.IsNull(imageID) {
		return false, ErrStorageInvalidID
	}

	query := bson.M{StorageKeyDeviceDeploymentAssignedImageId: imageID}

	if len(statuses) > 0 {
		query[StorageKeyDeviceDeploymentStatus] = bson.M{
			"$in": statuses,
		}
	}

	session := db.session.Copy()
	defer session.Close()

	// if found at least one then image in active deployment
	var tmp interface{}
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).One(&tmp); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// FindOldestDeploymentForDeviceIDWithStatuses find oldest deployment matching device id and one of specified statuses.
func (db *DataStoreMongo) FindOldestDeploymentForDeviceIDWithStatuses(ctx context.Context,
	deviceID string, statuses ...string) (*model.DeviceDeployment, error) {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) {
		return nil, ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	// Device should know only about deployments that are not finished
	query := bson.M{
		StorageKeyDeviceDeploymentDeviceId: deviceID,
		StorageKeyDeviceDeploymentStatus:   bson.M{"$in": statuses},
	}

	// Select only the oldest one that have not been finished yet.
	var deployment *model.DeviceDeployment
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).Sort("created").One(&deployment); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return deployment, nil
}

// FindAllDeploymentsForDeviceIDWithStatuses finds all deployments matching device id and one of specified statuses.
func (db *DataStoreMongo) FindAllDeploymentsForDeviceIDWithStatuses(ctx context.Context,
	deviceID string, statuses ...string) ([]model.DeviceDeployment, error) {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) {
		return nil, ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	// Device should know only about deployments that are not finished
	query := bson.M{
		StorageKeyDeviceDeploymentDeviceId: deviceID,
		StorageKeyDeviceDeploymentStatus: bson.M{
			"$in": statuses,
		},
	}

	var deployments []model.DeviceDeployment
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).All(&deployments); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return deployments, nil
}

func (db *DataStoreMongo) UpdateDeviceDeploymentStatus(ctx context.Context,
	deviceID string, deploymentID string, ddStatus model.DeviceDeploymentStatus) (string, error) {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) ||
		govalidator.IsNull(deploymentID) {
		return "", ErrStorageInvalidID
	}

	if ok, _ := govalidator.ValidateStruct(ddStatus); !ok {
		return "", ErrStorageInvalidInput
	}

	session := db.session.Copy()
	defer session.Close()

	// Device should know only about deployments that are not finished
	query := bson.M{
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
	}

	// update status field
	set := bson.M{
		StorageKeyDeviceDeploymentStatus: ddStatus.Status,
	}
	// and finish time if provided
	if ddStatus.FinishTime != nil {
		set[StorageKeyDeviceDeploymentFinished] = ddStatus.FinishTime
	}

	if ddStatus.SubState != nil {
		set[StorageKeyDeviceDeploymentSubState] = *ddStatus.SubState
	}

	update := bson.M{
		"$set": set,
	}

	var old model.DeviceDeployment

	// update and return the old status in one go
	change := mgo.Change{
		Update: update,
	}

	chi, err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).Apply(change, &old)

	if err != nil {
		if err == mgo.ErrNotFound {
			return "", ErrStorageNotFound
		}
		return "", err

	}

	if chi.Updated == 0 {
		return "", ErrStorageNotFound
	}

	return *old.Status, nil
}

func (db *DataStoreMongo) UpdateDeviceDeploymentLogAvailability(ctx context.Context,
	deviceID string, deploymentID string, log bool) error {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) ||
		govalidator.IsNull(deploymentID) {
		return ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	selector := bson.M{
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
	}

	update := bson.M{
		"$set": bson.M{
			StorageKeyDeviceDeploymentIsLogAvailable: log,
		},
	}

	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Update(selector, update); err != nil {
		if err == mgo.ErrNotFound {
			return ErrStorageNotFound
		}
		return err
	}

	return nil
}

// AssignArtifact assignes artifact to the device deployment
func (db *DataStoreMongo) AssignArtifact(ctx context.Context,
	deviceID string, deploymentID string, artifact *model.SoftwareImage) error {

	// Verify ID formatting
	if govalidator.IsNull(deviceID) ||
		govalidator.IsNull(deploymentID) {
		return ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	selector := bson.M{
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
	}

	update := bson.M{
		"$set": bson.M{
			StorageKeyDeviceDeploymentArtifact: artifact,
		},
	}

	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Update(selector, update); err != nil {
		if err == mgo.ErrNotFound {
			return ErrStorageNotFound
		}
		return err
	}

	return nil
}

func (db *DataStoreMongo) AggregateDeviceDeploymentByStatus(ctx context.Context,
	id string) (model.Stats, error) {

	if govalidator.IsNull(id) {
		return nil, ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	match := bson.M{
		"$match": bson.M{
			StorageKeyDeviceDeploymentDeploymentID: id,
		},
	}
	group := bson.M{
		"$group": bson.M{
			"_id": "$" + StorageKeyDeviceDeploymentStatus,
			"count": bson.M{
				"$sum": 1,
			},
		},
	}
	pipe := []bson.M{
		match,
		group,
	}
	var results []struct {
		Name  string `bson:"_id"`
		Count int
	}
	err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Pipe(&pipe).All(&results)
	if err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	raw := model.NewDeviceDeploymentStats()
	for _, res := range results {
		raw[res.Name] = res.Count
	}
	return raw, nil
}

//GetDeviceStatusesForDeployment retrieve device deployment statuses for a given deployment.
func (db *DataStoreMongo) GetDeviceStatusesForDeployment(ctx context.Context,
	deploymentID string) ([]model.DeviceDeployment, error) {

	session := db.session.Copy()
	defer session.Close()

	query := bson.M{
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
	}

	var statuses []model.DeviceDeployment

	err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).All(&statuses)
	if err != nil {
		return nil, err
	}

	return statuses, nil
}

// Returns true if deployment of ID `deploymentID` is assigned to device with ID
// `deviceID`, false otherwise. In case of errors returns false and an error
// that occurred
func (db *DataStoreMongo) HasDeploymentForDevice(ctx context.Context,
	deploymentID string, deviceID string) (bool, error) {

	session := db.session.Copy()
	defer session.Close()

	query := bson.M{
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
	}

	var dep model.DeviceDeployment
	err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).One(&dep)
	if err != nil {
		if err == mgo.ErrNotFound {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func (db *DataStoreMongo) GetDeviceDeploymentStatus(ctx context.Context,
	deploymentID string, deviceID string) (string, error) {

	session := db.session.Copy()
	defer session.Close()

	query := bson.M{
		StorageKeyDeviceDeploymentDeploymentID: deploymentID,
		StorageKeyDeviceDeploymentDeviceId:     deviceID,
	}

	var dep model.DeviceDeployment
	err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(query).One(&dep)
	if err != nil {
		if err == mgo.ErrNotFound {
			return "", nil
		} else {
			return "", err
		}
	}

	return *dep.Status, nil
}

func (db *DataStoreMongo) AbortDeviceDeployments(ctx context.Context,
	deploymentId string) error {

	if govalidator.IsNull(deploymentId) {
		return ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()
	selector := bson.M{
		"$and": []bson.M{
			{
				StorageKeyDeviceDeploymentDeploymentID: deploymentId,
			},
			{
				StorageKeyDeviceDeploymentStatus: bson.M{
					"$in": model.ActiveDeploymentStatuses(),
				},
			},
		},
	}

	update := bson.M{
		"$set": bson.M{
			StorageKeyDeviceDeploymentStatus: model.DeviceDeploymentStatusAborted,
		},
	}

	_, err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).UpdateAll(selector, update)

	if err == mgo.ErrNotFound {
		return ErrStorageInvalidID
	}

	return err
}

func (db *DataStoreMongo) DecommissionDeviceDeployments(ctx context.Context,
	deviceId string) error {

	if govalidator.IsNull(deviceId) {
		return ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()
	selector := bson.M{
		"$and": []bson.M{
			{
				StorageKeyDeviceDeploymentDeviceId: deviceId,
			},
			{
				StorageKeyDeviceDeploymentStatus: bson.M{
					"$in": model.ActiveDeploymentStatuses(),
				},
			},
		},
	}

	update := bson.M{
		"$set": bson.M{
			StorageKeyDeviceDeploymentStatus: model.DeviceDeploymentStatusDecommissioned,
		},
	}

	_, err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).UpdateAll(selector, update)

	return err
}

// deployments

func (db *DataStoreMongo) EnsureIndexing(ctx context.Context, session *mgo.Session) error {
	dataBase := mstore.DbFromContext(ctx, DatabaseName)

	return db.DoEnsureIndexing(dataBase, session)
}

func (db *DataStoreMongo) DoEnsureIndexing(dataBase string, session *mgo.Session) error {
	deploymentArtifactNameIndex := mgo.Index{
		Key:        StorageIndexes,
		Name:       IndexDeploymentArtifactNameStr,
		Background: false,
	}

	return session.DB(dataBase).
		C(CollectionDeployments).
		EnsureIndex(deploymentArtifactNameIndex)
}

func (db *DataStoreMongo) DoEnsureAdditionalIndexing(dataBase string, session *mgo.Session) error {
	deploymentDevicesStatusesIndex := mgo.Index{
		Key:        StatusIndexes,
		Name:       IndexDeploymentDeviceStatusesStr,
		Background: false,
	}

	err := session.DB(dataBase).
		C(CollectionDevices).
		EnsureIndex(deploymentDevicesStatusesIndex)

	if err != nil {
		return err
	}

	// IndexDeploymentDeviceIdStatusStr = "devicesIdWithStatus"
	// deviceID:1
	// status:1
	deploymentDevicesStatusIdIndex := mgo.Index{
		Key:        DeviceIDStatusIndexes,
		Name:       IndexDeploymentDeviceIdStatusStr,
		Background: false,
	}

	err = session.DB(dataBase).
		C(CollectionDevices).
		EnsureIndex(deploymentDevicesStatusIdIndex)

	if err != nil {
		return err
	}

	// IndexDeploymentDeviceDeploymentIdStr = "devicesDeploymentId"
	// deploymentid:1
	deploymentDeviceDeploymentIdIndex := mgo.Index{
		Key:        DeploymentIdIndexes,
		Name:       IndexDeploymentDeviceDeploymentIdStr,
		Background: false,
	}

	err = session.DB(dataBase).
		C(CollectionDevices).
		EnsureIndex(deploymentDeviceDeploymentIdIndex)

	if err != nil {
		return err
	}

	// IndexDeploymentStatusFinishedStr = "deploymentStatusFinished"
	// stats.downloading: 1
	// stats.installing: 1
	// stats.pending: 1
	// stats.rebooting: 1
	// created: -1
	deploymentStatusFinishedIndex := mgo.Index{
		Key:        DeploymentStatusFinishedIndex,
		Name:       IndexDeploymentStatusFinishedStr,
		Background: false,
	}

	err = session.DB(dataBase).
		C(CollectionDeployments).
		EnsureIndex(deploymentStatusFinishedIndex)

	if err != nil {
		return err
	}

	// IndexDeploymentStatusPendingStr = "deploymentStatusPending"
	// stats.aborted: 1
	// stats.already-installed: 1
	// stats.decommissioned: 1
	// stats.downloading: 1
	// stats.failure: 1
	// stats.installing: 1
	// stats.noartifact: 1
	// stats.rebooting: 1
	// stats.success: 1
	// created: -1
	deploymentStatusPendingIndex := mgo.Index{
		Key:        DeploymentStatusPendingIndex,
		Name:       IndexDeploymentStatusPendingStr,
		Background: false,
	}

	err = session.DB(dataBase).
		C(CollectionDeployments).
		EnsureIndex(deploymentStatusPendingIndex)

	if err != nil {
		return err
	}

	// IndexDeploymentCreatedStr = "deploymentCreated"
	// created: -1
	deploymentCreatedIndex := mgo.Index{
		Key:        DeploymentCreatedIndex,
		Name:       IndexDeploymentCreatedStr,
		Background: false,
	}

	err = session.DB(dataBase).
		C(CollectionDeployments).
		EnsureIndex(deploymentCreatedIndex)

	if err != nil {
		return err
	}

	// IndexDeploymentDeviceStatusRebootingStr = "deploymentsDeviceStatusRebooting"
	// stats.rebooting: 1
	deploymentDeviceStatusRebootingIndex := mgo.Index{
		Key:        DeploymentDeviceStatusRebootingIndex,
		Name:       IndexDeploymentDeviceStatusRebootingStr,
		Background: false,
	}

	err = session.DB(dataBase).
		C(CollectionDeployments).
		EnsureIndex(deploymentDeviceStatusRebootingIndex)

	if err != nil {
		return err
	}

	// IndexDeploymentDeviceStatusPendingStr = "deploymentsDeviceStatusPending"
	// stats.pending: 1
	deploymentDeviceStatusPendingIndex := mgo.Index{
		Key:        DeploymentDeviceStatusPendingIndex,
		Name:       IndexDeploymentDeviceStatusPendingStr,
		Background: false,
	}

	err = session.DB(dataBase).
		C(CollectionDeployments).
		EnsureIndex(deploymentDeviceStatusPendingIndex)

	if err != nil {
		return err
	}

	// IndexDeploymentDeviceStatusInstallingStr = "deploymentsDeviceStatusInstalling"
	// stats.installing: 1
	deploymentDeviceStatusInstallingIndex := mgo.Index{
		Key:        DeploymentDeviceStatusInstallingIndex,
		Name:       IndexDeploymentDeviceStatusInstallingStr,
		Background: false,
	}

	err = session.DB(dataBase).
		C(CollectionDeployments).
		EnsureIndex(deploymentDeviceStatusInstallingIndex)

	if err != nil {
		return err
	}

	// IndexDeploymentDeviceStatusFinishedStr = "deploymentsFinished"
	// finished: 1
	deploymentDeviceStatusFinishedIndex := mgo.Index{
		Key:        DeploymentDeviceStatusFinishedIndex,
		Name:       IndexDeploymentDeviceStatusFinishedStr,
		Background: false,
	}

	err = session.DB(dataBase).
		C(CollectionDeployments).
		EnsureIndex(deploymentDeviceStatusFinishedIndex)

	if err != nil {
		return err
	}

	return err
}

// return true if required indexing was set up
func (db *DataStoreMongo) hasIndexing(ctx context.Context, session *mgo.Session) bool {
	idxs, err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).Indexes()
	if err != nil {
		// check failed, assume indexing is not there
		return false
	}

	has := map[string]bool{}
	for _, idx := range idxs {
		for _, i := range idx.Key {
			has[i] = true
		}
	}

	for _, idx := range StorageIndexes {
		_, ok := has[idx]
		if !ok {
			return false
		}
	}

	return true
}

// Insert persists object
func (db *DataStoreMongo) InsertDeployment(ctx context.Context, deployment *model.Deployment) error {

	if deployment == nil {
		return ErrDeploymentStorageInvalidDeployment
	}

	if err := deployment.Validate(); err != nil {
		return err
	}

	session := db.session.Copy()
	defer session.Close()

	if err := db.EnsureIndexing(ctx, session); err != nil {
		return err
	}

	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).Insert(deployment); err != nil {
		return err
	}
	return nil
}

// Delete removed entry by ID
// Noop on ID not found
func (db *DataStoreMongo) DeleteDeployment(ctx context.Context, id string) error {

	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).RemoveId(id); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil
		}
		return err
	}

	return nil
}

func (db *DataStoreMongo) FindDeploymentByID(ctx context.Context, id string) (*model.Deployment, error) {

	if govalidator.IsNull(id) {
		return nil, ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	var deployment *model.Deployment
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).FindId(id).One(&deployment); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return deployment, nil
}

func (db *DataStoreMongo) FindUnfinishedByID(ctx context.Context,
	id string) (*model.Deployment, error) {

	if govalidator.IsNull(id) {
		return nil, ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	var deployment *model.Deployment
	filter := bson.M{
		"_id":                        id,
		StorageKeyDeploymentFinished: nil,
	}
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).Find(filter).One(&deployment); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil, nil
		}
		return nil, err
	}

	return deployment, nil
}

func (db *DataStoreMongo) DeviceCountByDeployment(ctx context.Context,
	id string) (int, error) {

	session := db.session.Copy()
	defer session.Close()

	filter := bson.M{
		"deploymentid": id,
	}

	deviceCount, err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDevices).Find(filter).Count()

	if err != nil {
		return 0, err
	}

	return deviceCount, nil
}

func (db *DataStoreMongo) UpdateStatsAndFinishDeployment(ctx context.Context,
	id string, stats model.Stats) error {

	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	deployment, err := model.NewDeployment()
	if err != nil {
		return errors.Wrap(err, "failed to create deployment")
	}

	deployment.Stats = stats
	var update bson.M
	if deployment.IsFinished() {
		now := time.Now()

		update = bson.M{
			"$set": bson.M{
				StorageKeyDeploymentStats:    stats,
				StorageKeyDeploymentFinished: &now,
			},
		}
	} else {
		update = bson.M{
			"$set": bson.M{
				StorageKeyDeploymentStats: stats,
			},
		}
	}

	err = session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).UpdateId(id, update)
	if err == mgo.ErrNotFound {
		return ErrStorageInvalidID
	}

	return err
}

func (db *DataStoreMongo) UpdateStats(ctx context.Context, id string,
	state_from, state_to string) error {

	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	if govalidator.IsNull(state_from) {
		return ErrStorageInvalidInput
	}

	if govalidator.IsNull(state_to) {
		return ErrStorageInvalidInput
	}

	// does not need any extra operations
	// following query won't handle this case well and increase the state_to value
	if state_from == state_to {
		return nil
	}

	session := db.session.Copy()
	defer session.Close()

	// note dot notation on embedded document
	update := bson.M{
		"$inc": bson.M{
			"stats." + state_from: -1,
			"stats." + state_to:   1,
		},
	}

	err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).UpdateId(id, update)

	if err == mgo.ErrNotFound {
		return ErrStorageInvalidID
	}

	return err
}

func buildStatusKey(status string) string {
	return StorageKeyDeploymentStats + "." + status
}

func buildStatusQuery(status model.StatusQuery) bson.M {

	gt0 := bson.M{"$gt": 0}
	eq0 := bson.M{"$eq": 0}
	notNull := bson.M{"$ne": nil}

	// empty query, catches StatusQueryAny
	stq := bson.M{}

	switch status {
	case model.StatusQueryInProgress:
		{
			// downloading, installing or rebooting are non 0, or
			// already-installed/success/failure/noimage >0 and pending > 0
			stq = bson.M{
				"$or": []bson.M{
					{
						buildStatusKey(model.DeviceDeploymentStatusDownloading): gt0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusInstalling): gt0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusRebooting): gt0,
					},
					{
						"$and": []bson.M{
							{
								buildStatusKey(model.DeviceDeploymentStatusPending): gt0,
							},
							{
								"$or": []bson.M{
									{
										buildStatusKey(model.DeviceDeploymentStatusAlreadyInst): gt0,
									},
									{
										buildStatusKey(model.DeviceDeploymentStatusSuccess): gt0,
									},
									{
										buildStatusKey(model.DeviceDeploymentStatusFailure): gt0,
									},
									{
										buildStatusKey(model.DeviceDeploymentStatusNoArtifact): gt0,
									},
								},
							},
						},
					},
				},
			}
		}

	case model.StatusQueryPending:
		{
			// all status counters, except for pending, are 0
			stq = bson.M{
				"$and": []bson.M{
					{
						buildStatusKey(model.DeviceDeploymentStatusDownloading): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusInstalling): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusRebooting): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusSuccess): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusAlreadyInst): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusAborted): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusDecommissioned): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusFailure): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusNoArtifact): eq0,
					},
					{
						buildStatusKey(model.DeviceDeploymentStatusPending): gt0,
					},
				},
			}
		}
	case model.StatusQueryFinished:
		{
			stq = bson.M{StorageKeyDeploymentFinished: notNull}
		}
	}

	return stq
}

func (db *DataStoreMongo) Find(ctx context.Context,
	match model.Query) ([]*model.Deployment, error) {

	session := db.session.Copy()
	defer session.Close()

	andq := []bson.M{}

	// build deployment by name part of the query
	if match.SearchText != "" {
		// we must have indexing for text search
		if !db.hasIndexing(ctx, session) {
			return nil, ErrDeploymentStorageCannotExecQuery
		}

		tq := bson.M{
			"$text": bson.M{
				"$search": match.SearchText,
			},
		}

		andq = append(andq, tq)
	}

	// build deployment by status part of the query
	if match.Status != model.StatusQueryAny {
		stq := buildStatusQuery(match.Status)
		andq = append(andq, stq)
	}

	query := bson.M{}
	if len(andq) != 0 {
		// use search criteria if any
		query = bson.M{
			"$and": andq,
		}
	}

	if match.CreatedAfter != nil && match.CreatedBefore != nil {
		query["created"] = bson.M{
			"$gte": match.CreatedAfter,
			"$lte": match.CreatedBefore,
		}
	} else if match.CreatedAfter != nil {
		query["created"] = bson.M{
			"$gte": match.CreatedAfter,
		}
	} else if match.CreatedBefore != nil {
		query["created"] = bson.M{
			"$lte": match.CreatedBefore,
		}
	}

	var deployment []*model.Deployment
	err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).
		Find(&query).Sort("-created").
		Skip(match.Skip).Limit(match.Limit).
		All(&deployment)

	if err != nil {
		return nil, err
	}

	return deployment, nil
}

func (db *DataStoreMongo) Finish(ctx context.Context, id string, when time.Time) error {
	if govalidator.IsNull(id) {
		return ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	// note dot notation on embedded document
	update := bson.M{
		"$set": bson.M{
			StorageKeyDeploymentFinished: &when,
		},
	}

	err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).UpdateId(id, update)

	if err == mgo.ErrNotFound {
		return ErrStorageInvalidID
	}

	return err
}

// ExistUnfinishedByArtifactId checks if there is an active deployment that uses
// given artifact
func (db *DataStoreMongo) ExistUnfinishedByArtifactId(ctx context.Context,
	id string) (bool, error) {

	if govalidator.IsNull(id) {
		return false, ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	var tmp interface{}
	query := bson.M{
		StorageKeyDeploymentFinished:  nil,
		StorageKeyDeploymentArtifacts: id,
	}
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).Find(query).One(&tmp); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ExistByArtifactId check if there is any deployment that uses give artifact
func (db *DataStoreMongo) ExistByArtifactId(ctx context.Context,
	id string) (bool, error) {

	if govalidator.IsNull(id) {
		return false, ErrStorageInvalidID
	}

	session := db.session.Copy()
	defer session.Close()

	var tmp interface{}
	query := bson.M{
		StorageKeyDeploymentArtifacts: id,
	}
	if err := session.DB(mstore.DbFromContext(ctx, DatabaseName)).
		C(CollectionDeployments).Find(query).One(&tmp); err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
